package edge

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// RegisterKubernetesClient registers a client with the Kubernetes API.
func RegisterKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", "/etc/kubernetes/admin.conf")
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}
