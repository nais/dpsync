package main

import (
	"context"
	"fmt"
	"github.com/nais/dpsync/controllers/reconcilers"
	dpsyncMetrics "github.com/nais/dpsync/pkg/metrics"
	naisv1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	naisv1Alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/conftools"
	schemeutil "github.com/nais/liberator/pkg/scheme"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
	"syscall"
)

type Config struct {
	LogFormat      string
	LogLevel       string
	MetricsAddress string
	SyncPeriod     string
}

const (
	MetricsAddress = "metrics-address"
	SyncPeriod     = "sync-period"
)

const (
	ExitOK = iota
	ExitConfig
	ExitRuntime
)

func init() {
	dpsyncMetrics.Register(metrics.Registry)
	// +kubebuilder:scaffold:scheme
}

func main() {
	config := &Config{
		LogLevel:       "debug",
		LogFormat:      "json",
		MetricsAddress: "/metrics",
		SyncPeriod:     "5",
	}

	log := log.New()
	ctx, cancel := context.WithCancel(context.Background())
	conftools.Initialize("dpsync")
	err := conftools.Load(config)
	if err != nil {
		log.Fatalf("Unable to read configurationn %w", err)
	}

	syncPeriod := viper.GetDuration(SyncPeriod)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		SyncPeriod:         &syncPeriod,
		Scheme:             schemeOrDie(),
		MetricsBindAddress: viper.GetString(MetricsAddress),
	})

	if err != nil {
		log.Errorf("creating manager: %s", err)
		os.Exit(ExitConfig)
	}

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

		for {
			select {
			case sig := <-signals:
				log.Infof("exiting due to signal: %s", strings.ToUpper(sig.String()))
				cancel()
				os.Exit(ExitOK)
			}
		}
	}()

	if err := setupReconcilers(mgr, log); err != nil {
		log.Fatalf("unable to set up reconciler: %s", err)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		log.Errorln(fmt.Errorf("manager stopped unexpectedly: %s", err))
		os.Exit(ExitRuntime)
	}

	log.Errorln("manager has stopped")
}

func setupReconcilers(mgr manager.Manager, log *log.Logger) error {
	applicationReconciler := reconcilers.NewApplicationReconciler(mgr, log)
	if err := applicationReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	naisjobReconciler := reconcilers.NewNaisJobReconciler(mgr, log)
	if err := naisjobReconciler.SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

func schemeOrDie() *runtime.Scheme {
	scheme, err := schemeutil.Scheme(
		naisv1Alpha1.AddToScheme,
		naisv1.AddToScheme,
		clientgoscheme.AddToScheme,
	)

	if err != nil {
		log.Fatalf("Setting up schemes for required CRDs: %v", err)
	}

	return scheme
}
