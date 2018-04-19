package edge

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Starts the process of reading Kubernetes services every read interval.
func (oe *OptikonEdge) startReadingServices() {

	// Starts a probe to call the read function every time interval.
	oe.svcReadProbe.Start(oe.svcReadInterval)

	// Register the probe function to read and update the service list.
	oe.svcReadProbe.Do(func() error {
		services, err := oe.clientset.Core().Services("").List(metaV1.ListOptions{})
		serviceDomains := make([]string, services.Size())
		for i, service := range services.Items {
			serviceDomains[i] = generateServiceDNS(&service)
		}
		oe.services.Overwrite(serviceDomains)
	})
}

// Generates a services DNS that looks like my-svc.my-namespace.svc.cluster.external
func generateServiceDNS(svc *v1.Service) string {
	return svc.GetName() + "." + svc.GetNamespace() + ".svc.cluster.external"
}

// Stops reading Kubernetes services into local state.
func (oe *OptikonEdge) stopReadingServices() {
	oe.svcReadProbe.Stop()
}
