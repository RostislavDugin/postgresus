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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasusv1alpha1 "github.com/databasus/databasus/operator/api/v1alpha1"
	dbclient "github.com/databasus/databasus/operator/internal/client"
)

const (
	databaseBackupFinalizer = "databasus.io/databasebackup-cleanup"
	statusPollInterval      = 60 * time.Second
)

type DatabaseBackupReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DatabasusClient *dbclient.DatabasusClient
}

// +kubebuilder:rbac:groups=databasus.databasus.io,resources=databasebackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=databasebackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=databasebackups/finalizers,verbs=update
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=storages,verbs=get;list;watch
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=notifiers,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *DatabaseBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var dbBackup databasusv1alpha1.DatabaseBackup
	if err := r.Get(ctx, req.NamespacedName, &dbBackup); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Handle deletion
	if !dbBackup.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&dbBackup, databaseBackupFinalizer) {
			if dbBackup.Status.DatabaseID != "" {
				logger.Info("deleting database from databasus", "database_id", dbBackup.Status.DatabaseID)

				if err := r.DatabasusClient.DeleteDatabase(ctx, dbBackup.Status.DatabaseID); err != nil {
					logger.Error(err, "failed to delete database from databasus")
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
			}

			controllerutil.RemoveFinalizer(&dbBackup, databaseBackupFinalizer)
			if err := r.Update(ctx, &dbBackup); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if missing
	if !controllerutil.ContainsFinalizer(&dbBackup, databaseBackupFinalizer) {
		controllerutil.AddFinalizer(&dbBackup, databaseBackupFinalizer)
		if err := r.Update(ctx, &dbBackup); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Resolve Storage reference
	storageID, err := r.resolveStorageRef(ctx, dbBackup.Namespace, dbBackup.Spec.Backup.StorageRef)
	if err != nil {
		logger.Info("waiting for storage dependency", "storage_ref", dbBackup.Spec.Backup.StorageRef, "error", err.Error())
		r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "DependencyNotReady", err.Error())
		_ = r.Status().Update(ctx, &dbBackup)

		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Resolve Notifier references
	notifierIDs, err := r.resolveNotifierRefs(ctx, dbBackup.Namespace, dbBackup.Spec.Database.NotifierRefs)
	if err != nil {
		logger.Info("waiting for notifier dependency", "error", err.Error())
		r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "DependencyNotReady", err.Error())
		_ = r.Status().Update(ctx, &dbBackup)

		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Resolve database password
	password, err := r.resolveDatabasePassword(ctx, &dbBackup)
	if err != nil {
		logger.Error(err, "failed to resolve database password")
		r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		_ = r.Status().Update(ctx, &dbBackup)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Build and send database request
	dbReq := r.buildDatabaseRequest(&dbBackup, password, notifierIDs)

	var dbResp *dbclient.DatabaseResponse

	if dbBackup.Status.DatabaseID == "" {
		dbResp, err = r.DatabasusClient.CreateDatabase(ctx, dbReq)
	} else {
		dbReq.ID = dbBackup.Status.DatabaseID
		dbResp, err = r.DatabasusClient.UpdateDatabase(ctx, dbReq)
	}

	if err != nil {
		logger.Error(err, "failed to sync database to databasus")
		r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "APISyncFailed", err.Error())
		_ = r.Status().Update(ctx, &dbBackup)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	dbBackup.Status.DatabaseID = dbResp.ID

	// Save backup config
	backupReq := r.buildBackupConfigRequest(&dbBackup, dbResp.ID, storageID)

	if _, err := r.DatabasusClient.SaveBackupConfig(ctx, backupReq); err != nil {
		logger.Error(err, "failed to sync backup config to databasus")
		r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "BackupConfigSyncFailed", err.Error())
		_ = r.Status().Update(ctx, &dbBackup)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Save healthcheck config if specified
	if dbBackup.Spec.Healthcheck != nil {
		hcReq := r.buildHealthcheckRequest(&dbBackup, dbResp.ID)

		if _, err := r.DatabasusClient.SaveHealthcheckConfig(ctx, hcReq); err != nil {
			logger.Error(err, "failed to sync healthcheck config to databasus")
			r.setCondition(&dbBackup, "Ready", metav1.ConditionFalse, "HealthcheckSyncFailed", err.Error())
			_ = r.Status().Update(ctx, &dbBackup)

			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Refresh status from databasus
	if refreshedDB, err := r.DatabasusClient.GetDatabase(ctx, dbResp.ID); err == nil && refreshedDB != nil {
		if refreshedDB.HealthStatus != nil {
			dbBackup.Status.HealthStatus = *refreshedDB.HealthStatus
		}

		if refreshedDB.LastBackupTime != nil {
			dbBackup.Status.LastBackupTime = &metav1.Time{Time: *refreshedDB.LastBackupTime}
		}

		if refreshedDB.LastBackupErrorMessage != nil {
			dbBackup.Status.LastBackupErrorMessage = *refreshedDB.LastBackupErrorMessage
		}
	}

	// Set success status
	dbBackup.Status.ObservedGeneration = dbBackup.Generation
	r.setCondition(&dbBackup, "Ready", metav1.ConditionTrue, "Synced", "DatabaseBackup synced to databasus")

	if err := r.Status().Update(ctx, &dbBackup); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("database backup synced successfully", "database_id", dbResp.ID)

	return ctrl.Result{RequeueAfter: statusPollInterval}, nil
}

func (r *DatabaseBackupReconciler) resolveStorageRef(ctx context.Context, namespace, storageRefName string) (string, error) {
	var storage databasusv1alpha1.Storage
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: storageRefName}, &storage); err != nil {
		return "", fmt.Errorf("storage %q not found: %w", storageRefName, err)
	}

	if storage.Status.ID == "" {
		return "", fmt.Errorf("storage %q not yet synced (no ID in status)", storageRefName)
	}

	readyCond := meta.FindStatusCondition(storage.Status.Conditions, "Ready")
	if readyCond == nil || readyCond.Status != metav1.ConditionTrue {
		return "", fmt.Errorf("storage %q is not ready", storageRefName)
	}

	return storage.Status.ID, nil
}

func (r *DatabaseBackupReconciler) resolveNotifierRefs(ctx context.Context, namespace string, notifierRefNames []string) ([]string, error) {
	var notifierIDs []string

	for _, name := range notifierRefNames {
		var notifier databasusv1alpha1.Notifier
		if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &notifier); err != nil {
			return nil, fmt.Errorf("notifier %q not found: %w", name, err)
		}

		if notifier.Status.ID == "" {
			return nil, fmt.Errorf("notifier %q not yet synced (no ID in status)", name)
		}

		readyCond := meta.FindStatusCondition(notifier.Status.Conditions, "Ready")
		if readyCond == nil || readyCond.Status != metav1.ConditionTrue {
			return nil, fmt.Errorf("notifier %q is not ready", name)
		}

		notifierIDs = append(notifierIDs, notifier.Status.ID)
	}

	return notifierIDs, nil
}

func (r *DatabaseBackupReconciler) resolveDatabasePassword(ctx context.Context, dbBackup *databasusv1alpha1.DatabaseBackup) (string, error) {
	var ref databasusv1alpha1.SecretKeyRef

	switch dbBackup.Spec.Database.Type {
	case databasusv1alpha1.DatabaseTypePostgres:
		if dbBackup.Spec.Database.Postgresql == nil {
			return "", fmt.Errorf("postgresql spec is required")
		}

		ref = dbBackup.Spec.Database.Postgresql.PasswordSecretRef

	case databasusv1alpha1.DatabaseTypeMysql:
		if dbBackup.Spec.Database.Mysql == nil {
			return "", fmt.Errorf("mysql spec is required")
		}

		ref = dbBackup.Spec.Database.Mysql.PasswordSecretRef

	case databasusv1alpha1.DatabaseTypeMariadb:
		if dbBackup.Spec.Database.Mariadb == nil {
			return "", fmt.Errorf("mariadb spec is required")
		}

		ref = dbBackup.Spec.Database.Mariadb.PasswordSecretRef

	case databasusv1alpha1.DatabaseTypeMongodb:
		if dbBackup.Spec.Database.Mongodb == nil {
			return "", fmt.Errorf("mongodb spec is required")
		}

		ref = dbBackup.Spec.Database.Mongodb.PasswordSecretRef

	default:
		return "", fmt.Errorf("unsupported database type: %s", dbBackup.Spec.Database.Type)
	}

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: dbBackup.Namespace, Name: ref.Name}, &secret); err != nil {
		return "", fmt.Errorf("secret %q not found: %w", ref.Name, err)
	}

	value, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %q", ref.Key, ref.Name)
	}

	return string(value), nil
}

func (r *DatabaseBackupReconciler) buildDatabaseRequest(dbBackup *databasusv1alpha1.DatabaseBackup, password string, notifierIDs []string) *dbclient.DatabaseRequest {
	req := &dbclient.DatabaseRequest{
		WorkspaceID: r.DatabasusClient.WorkspaceID(),
		Name:        dbBackup.Spec.Database.Name,
		Type:        string(dbBackup.Spec.Database.Type),
	}

	for _, id := range notifierIDs {
		req.Notifiers = append(req.Notifiers, dbclient.NotifierRef{ID: id})
	}

	switch dbBackup.Spec.Database.Type {
	case databasusv1alpha1.DatabaseTypePostgres:
		pgSpec := dbBackup.Spec.Database.Postgresql

		pgReq := &dbclient.PostgresqlRequest{
			Version:        pgSpec.Version,
			Host:           pgSpec.Host,
			Port:           pgSpec.Port,
			Username:       pgSpec.Username,
			Password:       password,
			IsHttps:        pgSpec.IsHttps,
			BackupType:     pgSpec.BackupType,
			IncludeSchemas: pgSpec.IncludeSchemas,
			CpuCount:       pgSpec.CpuCount,
		}

		if pgSpec.Database != "" {
			pgReq.Database = &pgSpec.Database
		}

		if pgReq.BackupType == "" {
			pgReq.BackupType = "PG_DUMP"
		}

		if pgReq.CpuCount == 0 {
			pgReq.CpuCount = 1
		}

		req.Postgresql = pgReq

	case databasusv1alpha1.DatabaseTypeMysql:
		mySpec := dbBackup.Spec.Database.Mysql

		req.Mysql = &dbclient.MysqlRequest{
			Version:  mySpec.Version,
			Host:     mySpec.Host,
			Port:     mySpec.Port,
			Username: mySpec.Username,
			Password: password,
			Database: mySpec.Database,
			IsHttps:  mySpec.IsHttps,
		}

	case databasusv1alpha1.DatabaseTypeMariadb:
		maSpec := dbBackup.Spec.Database.Mariadb

		req.Mariadb = &dbclient.MariadbRequest{
			Version:  maSpec.Version,
			Host:     maSpec.Host,
			Port:     maSpec.Port,
			Username: maSpec.Username,
			Password: password,
			Database: maSpec.Database,
			IsHttps:  maSpec.IsHttps,
		}

	case databasusv1alpha1.DatabaseTypeMongodb:
		mgSpec := dbBackup.Spec.Database.Mongodb

		mgReq := &dbclient.MongodbRequest{
			Version:            mgSpec.Version,
			Host:               mgSpec.Host,
			Port:               mgSpec.Port,
			Username:           mgSpec.Username,
			Password:           password,
			Database:           mgSpec.Database,
			AuthDatabase:       mgSpec.AuthDatabase,
			IsHttps:            mgSpec.IsHttps,
			IsSrv:              mgSpec.IsSrv,
			IsDirectConnection: mgSpec.IsDirectConnection,
			CpuCount:           mgSpec.CpuCount,
		}

		if mgReq.CpuCount == 0 {
			mgReq.CpuCount = 1
		}

		req.Mongodb = mgReq
	}

	return req
}

func (r *DatabaseBackupReconciler) buildBackupConfigRequest(dbBackup *databasusv1alpha1.DatabaseBackup, databaseID, storageID string) *dbclient.BackupConfigRequest {
	backup := dbBackup.Spec.Backup

	notificationTypes := make([]string, len(backup.SendNotificationsOn))
	for i, nt := range backup.SendNotificationsOn {
		notificationTypes[i] = string(nt)
	}

	encryption := string(backup.Encryption)
	if encryption == "" {
		encryption = "NONE"
	}

	return &dbclient.BackupConfigRequest{
		DatabaseID:          databaseID,
		IsBackupsEnabled:    backup.IsEnabled,
		RetentionPolicyType: string(backup.RetentionPolicy.Type),
		RetentionTimePeriod: backup.RetentionPolicy.TimePeriod,
		RetentionCount:      backup.RetentionPolicy.Count,
		RetentionGfsHours:   backup.RetentionPolicy.GfsHours,
		RetentionGfsDays:    backup.RetentionPolicy.GfsDays,
		RetentionGfsWeeks:   backup.RetentionPolicy.GfsWeeks,
		RetentionGfsMonths:  backup.RetentionPolicy.GfsMonths,
		RetentionGfsYears:   backup.RetentionPolicy.GfsYears,
		StorageID:           storageID,
		BackupInterval: &dbclient.IntervalRequest{
			Interval:       string(backup.Interval.Type),
			TimeOfDay:      backup.Interval.TimeOfDay,
			Weekday:        backup.Interval.Weekday,
			DayOfMonth:     backup.Interval.DayOfMonth,
			CronExpression: backup.Interval.CronExpression,
		},
		SendNotificationsOn: notificationTypes,
		IsRetryIfFailed:     backup.IsRetryIfFailed,
		MaxFailedTriesCount: backup.MaxFailedTriesCount,
		Encryption:          encryption,
	}
}

func (r *DatabaseBackupReconciler) buildHealthcheckRequest(dbBackup *databasusv1alpha1.DatabaseBackup, databaseID string) *dbclient.HealthcheckConfigRequest {
	hc := dbBackup.Spec.Healthcheck

	return &dbclient.HealthcheckConfigRequest{
		DatabaseID:                        databaseID,
		IsHealthcheckEnabled:              hc.IsEnabled,
		IsSentNotificationWhenUnavailable: hc.IsSentNotificationWhenUnavailable,
		IntervalMinutes:                   hc.IntervalMinutes,
		AttemptsBeforeConcideredAsDown:    hc.AttemptsBeforeConsideredAsDown,
		StoreAttemptsDays:                 hc.StoreAttemptsDays,
	}
}

func (r *DatabaseBackupReconciler) setCondition(dbBackup *databasusv1alpha1.DatabaseBackup, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&dbBackup.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: dbBackup.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *DatabaseBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasusv1alpha1.DatabaseBackup{}).
		Named("databasebackup").
		Complete(r)
}
