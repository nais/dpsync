package main

import (
	"context"
	"fmt"
	k8s2kafkametrics "github.com/nais/k8s-to-kafka/pkg/metrics"
	application_nais_io_v1_alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/liberator/pkg/conftools"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
	"syscall"
)

var scheme = runtime.NewScheme()

type Config struct {
	LogFormat      string
	LogLevel       string
	MetricsAddress string
	SyncPeriod     string
}

// Configuration options
const (
	KubernetesWriteRetryInterval = "kubernetes-write-retry-interval"
	LogFormat                    = "log-format"
	LogLevel                     = "log-level"
	MetricsAddress               = "metrics-address"
	SyncPeriod                   = "sync-period"
)

const (
	ExitOK = iota
	ExitController
	ExitConfig
	ExitRuntime
	ExitCredentialsManager
)

func main() {
	config := &Config{
		LogLevel:       "debug",
		LogFormat:      "json",
		MetricsAddress: "/metrics",
		SyncPeriod:     "5",
	}
	logger := log.New()
	ctx, cancel := context.WithCancel(context.Background())
	conftools.Initialize("k8s2kafka")
	err := conftools.Load(config)
	if err != nil {
		log.Fatalf("Unable to read configurationn %w", err)
	}

	syncPeriod := viper.GetDuration(SyncPeriod)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		SyncPeriod:         &syncPeriod,
		Scheme:             scheme,
		MetricsBindAddress: viper.GetString(MetricsAddress),
	})

	if err != nil {
		logger.Errorf("unable to set up aiven client: %s", err)
		os.Exit(ExitConfig)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctx.Done()); err != nil {
		logger.Errorln(fmt.Errorf("manager stopped unexpectedly: %s", err))
		os.Exit(ExitRuntime)
	}
	logger.Info("manager started")

	logger.Errorln(fmt.Errorf("manager has stopped"))

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

		for {
			select {
			case sig := <-signals:
				logger.Infof("exiting due to signal: %s", strings.ToUpper(sig.String()))
				cancel()
				os.Exit(ExitOK)
			}
		}
	}()

}

func init() {
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		log.Fatal(err)
	}

	err = application_nais_io_v1_alpha1.AddToScheme(scheme)
	if err != nil {
		log.Fatal(err)
	}

	k8s2kafkametrics.Register(metrics.Registry)
	// +kubebuilder:scaffold:scheme
}
