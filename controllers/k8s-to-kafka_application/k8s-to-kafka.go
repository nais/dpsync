package k8s_to_kafka_application

import (
	"context"
	"fmt"
	application_nais_io_v1_alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

const (
	requeueInterval    = time.Second * 10
	secretWriteTimeout = time.Second * 2

	rolloutComplete = "RolloutComplete"
	rolloutFailed   = "RolloutFailed"
)

func NewReconciler(mgr manager.Manager, logger *log.Logger) K8s2kafkaReconciler {
	return K8s2kafkaReconciler{
		Client:  mgr.GetClient(),
		Logger:  logger.WithFields(log.Fields{"component": "K8s2kafkaReconciler"}),
		Manager: mgr,
	}
}

type K8s2kafkaReconciler struct {
	client.Client
	Logger  *log.Entry
	Manager ctrl.Manager
}

func (r *K8s2kafkaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var application application_nais_io_v1_alpha1.Application

	logger := r.Logger.WithFields(log.Fields{
		"application": req.Name,
		"namespace":   req.Namespace,
	})

	logger.Infof("Processing request")

	fail := func(err error, requeue bool) (ctrl.Result, error) {
		if err != nil {
			logger.Error(err)
		}

		cr := ctrl.Result{}
		if requeue {
			cr.RequeueAfter = requeueInterval
		}
		return cr, nil
	}

	err := r.Get(ctx, req.NamespacedName, &application)
	switch {
	case errors.IsNotFound(err):
		return fail(fmt.Errorf("resource deleted from cluster; noop"), false)
	case err != nil:
		return fail(fmt.Errorf("unable to retrieve resource from cluster: %s", err), true)
	}
	logger.Infof("Got ourselves a spanking fine app: %v", application.Name)

	return ctrl.Result{}, nil
}
