package namespace

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReadAnnotation(namespace, annotationKey string, cli client.Client, ctx context.Context) (string, error) {
	key := client.ObjectKey{Name: namespace}
	ns := corev1.Namespace{}
	err := cli.Get(ctx, key, &ns)
	if err != nil {
		log.Errorf("reading namespace: %v", err)
		return "", err
	}

	annotationValue := ns.Annotations[annotationKey]
	if annotationValue == "" {
		return "", fmt.Errorf("annotation %s not found on namespace %s", annotationKey, namespace)
	}

	return annotationValue, nil
}
