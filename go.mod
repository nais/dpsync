module github.com/nais/k8s-to-kafka

go 1.16

require (
	github.com/nais/liberator v0.0.0-20210607113907-4b4cf7624135
	github.com/prometheus/client_golang v1.0.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.3.2
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
