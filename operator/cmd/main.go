package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	databasusv1alpha1 "github.com/databasus/databasus/operator/api/v1alpha1"
	dbclient "github.com/databasus/databasus/operator/internal/client"
	"github.com/databasus/databasus/operator/internal/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(databasusv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true, "If set, the metrics endpoint is served securely via HTTPS.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "", "The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Databasus API URL from env
	databasusAPIURL := os.Getenv("DATABASUS_API_URL")
	if databasusAPIURL == "" {
		databasusAPIURL = "http://databasus-service.databasus.svc.cluster.local:4005"
	}

	// Credentials secret name and namespace
	credentialsSecretName := os.Getenv("DATABASUS_CREDENTIALS_SECRET")
	if credentialsSecretName == "" {
		credentialsSecretName = "databasus-operator-credentials"
	}

	credentialsSecretNamespace := os.Getenv("DATABASUS_CREDENTIALS_NAMESPACE")
	if credentialsSecretNamespace == "" {
		credentialsSecretNamespace = os.Getenv("POD_NAMESPACE")
	}
	if credentialsSecretNamespace == "" {
		credentialsSecretNamespace = "databasus"
	}

	// Disable HTTP/2 by default due to CVEs
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		})
	}

	webhookServerOptions := webhook.Options{TLSOpts: tlsOpts}
	if len(webhookCertPath) > 0 {
		webhookServerOptions.CertDir = webhookCertPath
		webhookServerOptions.CertName = webhookCertName
		webhookServerOptions.KeyName = webhookCertKey
	}

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	if len(metricsCertPath) > 0 {
		metricsServerOptions.CertDir = metricsCertPath
		metricsServerOptions.CertName = metricsCertName
		metricsServerOptions.KeyName = metricsCertKey
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhook.NewServer(webhookServerOptions),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9b691b66.databasus.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Read credentials from K8s Secret
	setupLog.Info("reading credentials", "secret", credentialsSecretName, "namespace", credentialsSecretNamespace)

	apiClient, err := authenticateFromSecret(mgr, databasusAPIURL, credentialsSecretNamespace, credentialsSecretName)
	if err != nil {
		setupLog.Error(err, "failed to authenticate with databasus")
		os.Exit(1)
	}

	setupLog.Info("authenticated with databasus",
		"url", databasusAPIURL,
		"workspace_id", apiClient.WorkspaceID(),
	)

	if err := (&controller.StorageReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		DatabasusClient: apiClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Storage")
		os.Exit(1)
	}

	if err := (&controller.NotifierReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		DatabasusClient: apiClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Notifier")
		os.Exit(1)
	}

	if err := (&controller.DatabaseBackupReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		DatabasusClient: apiClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DatabaseBackup")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// authenticateFromSecret reads the credentials Secret and authenticates with databasus.
// Secret expected keys:
//   - email: databasus user email
//   - password: databasus user password
//   - workspaceName (optional): workspace to use, defaults to first available
//   - workspaceId (optional): explicit workspace UUID (takes precedence over workspaceName)
func authenticateFromSecret(mgr ctrl.Manager, apiURL, namespace, secretName string) (*dbclient.DatabasusClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a direct client (the manager cache isn't started yet)
	directClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		return nil, fmt.Errorf("failed to create direct client: %w", err)
	}

	var secret corev1.Secret
	if err := directClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to read credentials secret %s/%s: %w", namespace, secretName, err)
	}

	email := string(secret.Data["email"])
	password := string(secret.Data["password"])

	if email == "" || password == "" {
		return nil, fmt.Errorf("credentials secret must contain 'email' and 'password' keys")
	}

	// Authenticate
	token, err := dbclient.SignIn(ctx, apiURL, email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	// Resolve workspace
	workspaceID := string(secret.Data["workspaceId"])

	if workspaceID == "" {
		workspaceName := string(secret.Data["workspaceName"])

		tempClient := dbclient.New(dbclient.Config{
			BaseURL:     apiURL,
			Token:       token,
			WorkspaceID: "",
		})

		resolvedID, err := tempClient.ResolveWorkspace(ctx, workspaceName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve workspace: %w", err)
		}

		workspaceID = resolvedID
	}

	apiClient := dbclient.New(dbclient.Config{
		BaseURL:     apiURL,
		Token:       token,
		WorkspaceID: workspaceID,
	})

	// Verify connectivity
	if err := apiClient.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("health check failed after auth: %w", err)
	}

	return apiClient, nil
}
