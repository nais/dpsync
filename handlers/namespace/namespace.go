package namespace

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct{}

func (c *Handler) ReadProject(namespace string, cli client.Client, ctx context.Context) (string, error) {
	ns := corev1.NamespaceList{}
	err := cli.List(ctx, &ns, client.InNamespace(namespace))
	if err != nil {
		log.Errorf("reading namespace: %v", err)
		return "", err
	}
	fmt.Println(ns.Items[0].Annotations)
	return "", nil
}
