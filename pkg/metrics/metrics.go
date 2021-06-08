package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	Namespace = "aura"

	LabelOperation    = "operation"
	LabelNamespace    = "namespace"
	LabelPool         = "pool"
	LabelResourceType = "resource_type"
	LabelStatus       = "status"
	LabelSyncState    = "synchronization_state"
)

var (
	KubernetesResourcesWritten = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "kubernetes_resources_written",
		Namespace: Namespace,
		Help:      "number of kubernetes resources written to the cluster",
	}, []string{LabelNamespace, LabelResourceType})

	KubernetesResourcesDeleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "kubernetes_resources_deleted",
		Namespace: Namespace,
		Help:      "number of kubernetes resources deleted from the cluster",
	}, []string{LabelNamespace, LabelResourceType})
)

func Register(registry prometheus.Registerer) {
	registry.MustRegister(
		KubernetesResourcesWritten,
		KubernetesResourcesDeleted,
	)
}
