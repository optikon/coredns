package central

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RegisterKubernetesClient registers an in-cluster client with the Kubernetes API.
func RegisterKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return clientset, nil
}
