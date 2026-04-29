package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasusv1alpha1 "github.com/databasus/databasus/operator/api/v1alpha1"
	dbclient "github.com/databasus/databasus/operator/internal/client"
)

const storageFinalizer = "databasus.io/storage-cleanup"

type StorageReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	DatabasusClient *dbclient.DatabasusClient
}

// +kubebuilder:rbac:groups=databasus.databasus.io,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=storages/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *StorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var storage databasusv1alpha1.Storage
	if err := r.Get(ctx, req.NamespacedName, &storage); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Handle deletion
	if !storage.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&storage, storageFinalizer) {
			if storage.Status.ID != "" {
				logger.Info("deleting storage from databasus", "storage_id", storage.Status.ID)

				if err := r.DatabasusClient.DeleteStorage(ctx, storage.Status.ID); err != nil {
					logger.Error(err, "failed to delete storage from databasus")
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
			}

			controllerutil.RemoveFinalizer(&storage, storageFinalizer)
			if err := r.Update(ctx, &storage); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if missing — return early so re-reconcile uses fresh resourceVersion
	if !controllerutil.ContainsFinalizer(&storage, storageFinalizer) {
		controllerutil.AddFinalizer(&storage, storageFinalizer)
		return ctrl.Result{}, r.Update(ctx, &storage)
	}

	// Build API request
	apiReq, err := r.buildStorageRequest(ctx, &storage)
	if err != nil {
		logger.Error(err, "failed to build storage request")
		r.setCondition(&storage, "Ready", metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		_ = r.Status().Update(ctx, &storage)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Set ID for update
	apiReq.ID = storage.Status.ID

	// Save to databasus
	resp, err := r.DatabasusClient.SaveStorage(ctx, apiReq)
	if err != nil {
		logger.Error(err, "failed to save storage to databasus")
		r.setCondition(&storage, "Ready", metav1.ConditionFalse, "APISyncFailed", err.Error())
		_ = r.Status().Update(ctx, &storage)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status
	storage.Status.ID = resp.ID
	storage.Status.ObservedGeneration = storage.Generation
	r.setCondition(&storage, "Ready", metav1.ConditionTrue, "Synced", "Storage synced to databasus")

	if err := r.Status().Update(ctx, &storage); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("storage synced successfully", "storage_id", resp.ID)

	return ctrl.Result{}, nil
}

func (r *StorageReconciler) buildStorageRequest(ctx context.Context, storage *databasusv1alpha1.Storage) (*dbclient.StorageRequest, error) {
	req := &dbclient.StorageRequest{
		WorkspaceID: r.DatabasusClient.WorkspaceID(),
		Type:        string(storage.Spec.Type),
		Name:        storage.Spec.Name,
	}

	switch storage.Spec.Type {
	case databasusv1alpha1.StorageTypeS3:
		if storage.Spec.S3 == nil {
			return nil, fmt.Errorf("s3 spec is required for S3 storage type")
		}

		accessKey, err := r.resolveSecretRef(ctx, storage.Namespace, storage.Spec.S3.AccessKeySecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve s3 access key: %w", err)
		}

		secretKey, err := r.resolveSecretRef(ctx, storage.Namespace, storage.Spec.S3.SecretKeySecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve s3 secret key: %w", err)
		}

		req.S3Storage = &dbclient.S3Request{
			S3Bucket:                storage.Spec.S3.Bucket,
			S3Region:                storage.Spec.S3.Region,
			S3AccessKey:             accessKey,
			S3SecretKey:             secretKey,
			S3Endpoint:              storage.Spec.S3.Endpoint,
			S3Prefix:                storage.Spec.S3.Prefix,
			S3UseVirtualHostedStyle: storage.Spec.S3.IsUseVirtualHostedStyle,
			SkipTLSVerify:           storage.Spec.S3.IsSkipTLSVerify,
			S3StorageClass:          storage.Spec.S3.StorageClass,
		}

	case databasusv1alpha1.StorageTypeSFTP:
		if storage.Spec.SFTP == nil {
			return nil, fmt.Errorf("sftp spec is required for SFTP storage type")
		}

		sftpReq := &dbclient.SFTPRequest{
			Host:                storage.Spec.SFTP.Host,
			Port:                storage.Spec.SFTP.Port,
			Username:            storage.Spec.SFTP.Username,
			Path:                storage.Spec.SFTP.Path,
			IsSkipHostKeyVerify: storage.Spec.SFTP.IsSkipHostKeyVerify,
		}

		if storage.Spec.SFTP.PasswordSecretRef != nil {
			password, err := r.resolveSecretRef(ctx, storage.Namespace, *storage.Spec.SFTP.PasswordSecretRef)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve sftp password: %w", err)
			}

			sftpReq.Password = password
		}

		if storage.Spec.SFTP.PrivateKeySecretRef != nil {
			privateKey, err := r.resolveSecretRef(ctx, storage.Namespace, *storage.Spec.SFTP.PrivateKeySecretRef)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve sftp private key: %w", err)
			}

			sftpReq.PrivateKey = privateKey
		}

		req.SFTPStorage = sftpReq
	}

	return req, nil
}

func (r *StorageReconciler) resolveSecretRef(ctx context.Context, namespace string, ref databasusv1alpha1.SecretKeyRef) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", fmt.Errorf("secret %q not found: %w", ref.Name, err)
	}

	value, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %q", ref.Key, ref.Name)
	}

	return string(value), nil
}

func (r *StorageReconciler) setCondition(storage *databasusv1alpha1.Storage, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&storage.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: storage.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *StorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasusv1alpha1.Storage{}).
		Named("storage").
		Complete(r)
}
