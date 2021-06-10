package reconcilers

import (
	"context"
	"fmt"
	"github.com/nais/dpsync/handlers/namespace"
	"github.com/nais/dpsync/pkg/dataproduct"
	naisV1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewNaisJobReconciler(mgr manager.Manager, logger *log.Logger) NaisjobReconciler {
	return NaisjobReconciler{
		Client:   mgr.GetClient(),
		Logger:   logger.WithFields(log.Fields{"component": "dpsync"}),
		Manager:  mgr,
		Recorder: mgr.GetEventRecorderFor("dpsync"),
	}
}

type NaisjobReconciler struct {
	client.Client
	Logger   *log.Entry
	Manager  ctrl.Manager
	Recorder record.EventRecorder
}

func (r *NaisjobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&naisV1.Naisjob{}).
		Complete(r)
}

func (r *NaisjobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var naisjob naisV1.Naisjob

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

	err := r.Get(ctx, req.NamespacedName, &naisjob)
	switch {
	case errors.IsNotFound(err):
		return fail(fmt.Errorf("resource deleted from cluster; noop"), false)
	case err != nil:
		return fail(fmt.Errorf("unable to retrieve resource from cluster: %s", err), true)
	}

	if len(naisjob.Annotations["dp_name"]) > 0 && len(naisjob.Annotations["dp_description"]) > 0 {
		dp := dataproduct.Dataproduct{}
		dp.Team = naisjob.Labels["team"]
		dp.Name = naisjob.Annotations["dp_name"]
		dp.Description = naisjob.Annotations["dp_description"]

		if len(naisjob.Spec.GCP.Buckets) > 0 {
			projectId, err := namespace.ReadAnnotation(naisjob.Namespace, "cnrm.cloud.google.com/project-id", r, context.Background())
			if err != nil {
				r.Recorder.Eventf(&naisjob, "Warning", "FailedDataProductRegistration", "Getting GoogleProjectID: %s", err)
				logger.Errorf("reading projectID from namespace: %v", err)
				return ctrl.Result{}, nil
			}
			datastore := map[string]string{}
			datastore["type"] = "bucket"
			datastore["bucket_id"] = naisjob.Spec.GCP.Buckets[0].Name
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
