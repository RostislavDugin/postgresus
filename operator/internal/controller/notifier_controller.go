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

const notifierFinalizer = "databasus.io/notifier-cleanup"

type NotifierReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DatabasusClient *dbclient.DatabasusClient
}

// +kubebuilder:rbac:groups=databasus.databasus.io,resources=notifiers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=notifiers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=databasus.databasus.io,resources=notifiers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *NotifierReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	var notifier databasusv1alpha1.Notifier
	if err := r.Get(ctx, req.NamespacedName, &notifier); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Handle deletion
	if !notifier.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&notifier, notifierFinalizer) {
			if notifier.Status.ID != "" {
				logger.Info("deleting notifier from databasus", "notifier_id", notifier.Status.ID)

				if err := r.DatabasusClient.DeleteNotifier(ctx, notifier.Status.ID); err != nil {
					logger.Error(err, "failed to delete notifier from databasus")
					return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
				}
			}

			controllerutil.RemoveFinalizer(&notifier, notifierFinalizer)
			if err := r.Update(ctx, &notifier); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// Add finalizer if missing — return early so re-reconcile uses fresh resourceVersion
	if !controllerutil.ContainsFinalizer(&notifier, notifierFinalizer) {
		controllerutil.AddFinalizer(&notifier, notifierFinalizer)
		return ctrl.Result{}, r.Update(ctx, &notifier)
	}

	// Build API request
	apiReq, err := r.buildNotifierRequest(ctx, &notifier)
	if err != nil {
		logger.Error(err, "failed to build notifier request")
		r.setCondition(&notifier, "Ready", metav1.ConditionFalse, "SecretResolutionFailed", err.Error())
		_ = r.Status().Update(ctx, &notifier)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Set ID for update
	apiReq.ID = notifier.Status.ID

	// Save to databasus
	resp, err := r.DatabasusClient.SaveNotifier(ctx, apiReq)
	if err != nil {
		logger.Error(err, "failed to save notifier to databasus")
		r.setCondition(&notifier, "Ready", metav1.ConditionFalse, "APISyncFailed", err.Error())
		_ = r.Status().Update(ctx, &notifier)

		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status
	notifier.Status.ID = resp.ID
	notifier.Status.ObservedGeneration = notifier.Generation
	r.setCondition(&notifier, "Ready", metav1.ConditionTrue, "Synced", "Notifier synced to databasus")

	if err := r.Status().Update(ctx, &notifier); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("notifier synced successfully", "notifier_id", resp.ID)

	return ctrl.Result{}, nil
}

func (r *NotifierReconciler) buildNotifierRequest(ctx context.Context, notifier *databasusv1alpha1.Notifier) (*dbclient.NotifierRequest, error) {
	req := &dbclient.NotifierRequest{
		WorkspaceID:  r.DatabasusClient.WorkspaceID(),
		Name:         notifier.Spec.Name,
		NotifierType: string(notifier.Spec.Type),
	}

	switch notifier.Spec.Type {
	case databasusv1alpha1.NotifierTypeDiscord:
		if notifier.Spec.Discord == nil {
			return nil, fmt.Errorf("discord spec is required for DISCORD notifier type")
		}

		webhookURL, err := r.resolveSecretRef(ctx, notifier.Namespace, notifier.Spec.Discord.WebhookURLSecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve discord webhook URL: %w", err)
		}

		req.DiscordNotifier = &dbclient.DiscordRequest{
			ChannelWebhookURL: webhookURL,
		}

	case databasusv1alpha1.NotifierTypeTelegram:
		if notifier.Spec.Telegram == nil {
			return nil, fmt.Errorf("telegram spec is required for TELEGRAM notifier type")
		}

		botToken, err := r.resolveSecretRef(ctx, notifier.Namespace, notifier.Spec.Telegram.BotTokenSecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve telegram bot token: %w", err)
		}

		req.TelegramNotifier = &dbclient.TelegramRequest{
			BotToken:     botToken,
			TargetChatID: notifier.Spec.Telegram.TargetChatID,
			ThreadID:     notifier.Spec.Telegram.ThreadID,
		}

	case databasusv1alpha1.NotifierTypeSlack:
		if notifier.Spec.Slack == nil {
			return nil, fmt.Errorf("slack spec is required for SLACK notifier type")
		}

		botToken, err := r.resolveSecretRef(ctx, notifier.Namespace, notifier.Spec.Slack.BotTokenSecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve slack bot token: %w", err)
		}

		req.SlackNotifier = &dbclient.SlackRequest{
			BotToken:     botToken,
			TargetChatID: notifier.Spec.Slack.TargetChatID,
		}

	case databasusv1alpha1.NotifierTypeEmail:
		if notifier.Spec.Email == nil {
			return nil, fmt.Errorf("email spec is required for EMAIL notifier type")
		}

		emailReq := &dbclient.EmailRequest{
			TargetEmail:          notifier.Spec.Email.TargetEmail,
			SMTPHost:             notifier.Spec.Email.SMTPHost,
			SMTPPort:             notifier.Spec.Email.SMTPPort,
			SMTPUser:             notifier.Spec.Email.SMTPUser,
			From:                 notifier.Spec.Email.From,
			IsInsecureSkipVerify: notifier.Spec.Email.IsInsecureSkipVerify,
		}

		if notifier.Spec.Email.SMTPPasswordSecretRef != nil {
			password, err := r.resolveSecretRef(ctx, notifier.Namespace, *notifier.Spec.Email.SMTPPasswordSecretRef)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve smtp password: %w", err)
			}

			emailReq.SMTPPassword = password
		}

		req.EmailNotifier = emailReq

	case databasusv1alpha1.NotifierTypeWebhook:
		if notifier.Spec.Webhook == nil {
			return nil, fmt.Errorf("webhook spec is required for WEBHOOK notifier type")
		}

		webhookReq := &dbclient.WebhookRequest{
			WebhookURL:    notifier.Spec.Webhook.WebhookURL,
			WebhookMethod: notifier.Spec.Webhook.WebhookMethod,
			BodyTemplate:  notifier.Spec.Webhook.BodyTemplate,
		}

		for _, header := range notifier.Spec.Webhook.Headers {
			headerReq := dbclient.WebhookHeaderRequest{Key: header.Key}

			if header.ValueSecretRef != nil {
				value, err := r.resolveSecretRef(ctx, notifier.Namespace, *header.ValueSecretRef)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve webhook header %q: %w", header.Key, err)
				}

				headerReq.Value = value
			} else {
				headerReq.Value = header.Value
			}

			webhookReq.Headers = append(webhookReq.Headers, headerReq)
		}

		req.WebhookNotifier = webhookReq

	case databasusv1alpha1.NotifierTypeTeams:
		if notifier.Spec.Teams == nil {
			return nil, fmt.Errorf("teams spec is required for TEAMS notifier type")
		}

		webhookURL, err := r.resolveSecretRef(ctx, notifier.Namespace, notifier.Spec.Teams.WebhookURLSecretRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve teams webhook URL: %w", err)
		}

		req.TeamsNotifier = &dbclient.TeamsRequest{
			ChannelWebhookURL: webhookURL,
		}
	}

	return req, nil
}

func (r *NotifierReconciler) resolveSecretRef(ctx context.Context, namespace string, ref databasusv1alpha1.SecretKeyRef) (string, error) {
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

func (r *NotifierReconciler) setCondition(notifier *databasusv1alpha1.Notifier, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&notifier.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: notifier.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

func (r *NotifierReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasusv1alpha1.Notifier{}).
		Named("notifier").
		Complete(r)
}
