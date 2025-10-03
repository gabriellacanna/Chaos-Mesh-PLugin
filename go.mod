module github.com/gabriellacanna/chaos-mesh-plugin

go 1.21

replace github.com/argoproj/argo-rollouts => github.com/argoproj/argo-rollouts v1.6.0

require (
	github.com/argoproj/argo-rollouts v0.0.0-00010101000000-000000000000
	github.com/hashicorp/go-plugin v1.4.10
	github.com/sirupsen/logrus v1.9.3
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	sigs.k8s.io/yaml v1.3.0
)