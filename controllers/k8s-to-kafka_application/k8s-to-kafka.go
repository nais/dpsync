package k8s_to_kafka_application

import (
	"context"
	"fmt"
	"github.com/nais/k8s-to-kafka/handlers/namespace"
	application_nais_io_v1_alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
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
	requeueInterval    = time.Second * 10
	secretWriteTimeout = time.Second * 2

	rolloutComplete = "RolloutComplete"
	rolloutFailed   = "RolloutFailed"
)

func NewReconciler(mgr manager.Manager, logger *log.Logger) K8s2kafkaReconciler {
	return K8s2kafkaReconciler{
		Client:   mgr.GetClient(),
		Logger:   logger.WithFields(log.Fields{"component": "dpsync"}),
		Manager:  mgr,
		Recorder: mgr.GetEventRecorderFor("dpsync"),
	}
}

type Dataproduct struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Team        string            `json:"team"`
	Datastore   map[string]string `json:"datastore"`
}

type K8s2kafkaReconciler struct {
	client.Client
	Logger   *log.Entry
	Manager  ctrl.Manager
	Recorder record.EventRecorder
}

func (r *K8s2kafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&application_nais_io_v1_alpha1.Application{}).
		Complete(r)
}

func (r *K8s2kafkaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
	ctx := context.Background()

	err := r.Get(ctx, req.NamespacedName, &application)
	switch {
	case errors.IsNotFound(err):
		return fail(fmt.Errorf("resource deleted from cluster; noop"), false)
	case err != nil:
		return fail(fmt.Errorf("unable to retrieve resource from cluster: %s", err), true)
	}
	if len(application.Annotations["dp_name"]) > 0 && len(application.Annotations["dp_description"]) > 0 {
		dp := Dataproduct{}
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
