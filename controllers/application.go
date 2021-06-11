package controllers

import (
	"context"
	"fmt"
	"github.com/nais/dpsync/handlers/namespace"
	"github.com/nais/dpsync/pkg/dataproduct"
	"github.com/nais/dpsync/pkg/kafka"
	naisV1Alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"time"
)

const (
	requeueInterval = time.Second * 10
)

func NewApplicationReconciler(mgr manager.Manager, logger *log.Logger, producer *kafka.Producer) ApplicationReconciler {
	return ApplicationReconciler{
		Client: mgr.GetClient(),
		Logger: logger.WithFields(
			log.Fields{
				"component":  "dpsync",
				"reconciler": "application",
			}),
		Manager:  mgr,
		Recorder: mgr.GetEventRecorderFor("dpsync"),
		Producer: producer,
	}
}

type ApplicationReconciler struct {
	client.Client
	Logger   *log.Entry
	Manager  ctrl.Manager
	Recorder record.EventRecorder
	Producer *kafka.Producer
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisV1Alpha1.Application{}).
		Complete(r)
}

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var application naisV1Alpha1.Application

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
	if len(application.Annotations["dp_name"]) > 0 && len(application.Annotations["dp_description"]) > 0 {
		dp := dataproduct.Dataproduct{}
		dp.Team = application.Labels["team"]
		dp.Name = application.Annotations["dp_name"]
		dp.Description = application.Annotations["dp_description"]

		if len(application.Spec.GCP.Buckets) > 0 {
			projectId, err := namespace.ReadAnnotation(application.Namespace, "cnrm.cloud.google.com/project-id", r, context.Background())
			if err != nil {
				r.Recorder.Eventf(&application, "Warning", "FailedDataProductRegistration", "Getting GoogleProjectID: %s", err)
				logger.Errorf("reading projectID from namespace: %v", err)
				return ctrl.Result{}, nil
			}
			datastore := map[string]string{}
			datastore["type"] = "bucket"
			datastore["bucket_id"] = application.Spec.GCP.Buckets[0].Name
			datastore["project_id"] = projectId
			dp.Datastore = datastore
		}

		dpJson, err := json.Marshal(&dp)
		if err != nil {
			fmt.Errorf("marshalling dataproduct to json: %v", err)
		}

		logger.Infof("this is our dp: %s", dpJson)

	}

	return ctrl.Result{}, nil
}
