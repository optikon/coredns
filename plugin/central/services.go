package central

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Starts the process of reading Kubernetes services every read interval.
func (oc *OptikonCentral) startReadingServices() {

	// Starts a probe to call the read function every time interval.
	oc.svcReadProbe.Start(oc.svcReadInterval)

	// Register the probe function to read and update the service list.
	oc.svcReadProbe.Do(func() error {
		services, err := oc.clientset.Core().Services("").List(metaV1.ListOptions{})
		if err != nil {
			return err
		}
		serviceDomains := make([]string, services.Size())
		for i, service := range services.Items {
			serviceDomains[i] = generateServiceDNS(&service)
		}
		oc.table.Update(oc.ip, oc.lon, oc.lat, serviceDomains)
		return nil
	})
}

// Generates a services DNS that looks like my-svc.my-namespace.svc.cluster.external
func generateServiceDNS(svc *v1.Service) string {
	return svc.GetName() + "." + svc.GetNamespace() + ".svc.cluster.external"
}

// Stops reading Kubernetes services into local state.
func (oc *OptikonCentral) stopReadingServices() {
	oc.svcReadProbe.Stop()
}
