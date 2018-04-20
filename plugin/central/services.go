package central

import (
	"time"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Starts the process of reading Kubernetes services every read interval.
func (oc *OptikonCentral) startReadingServices() {
	ticker := time.NewTicker(oc.svcReadInterval)
	oc.svcReadStopper = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				services, err := oc.clientset.CoreV1().Services("").List(metaV1.ListOptions{})
				if err != nil {
					continue
				}
				serviceDomains := make([]string, len(services.Items))
				for i, service := range services.Items {
					serviceDomains[i] = generateServiceDNS(&service)
				}
				oc.table.Update(oc.ip, oc.lon, oc.lat, serviceDomains)
			case <-oc.svcReadStopper:
				ticker.Stop()
				return
			}
		}
	}()
}

// Generates a services DNS that looks like my-svc.my-namespace.svc.cluster.external
func generateServiceDNS(svc *v1.Service) string {
	return svc.GetName() + "." + svc.GetNamespace() + ".svc.cluster.external"
}

// Stops reading Kubernetes services into local state.
func (oc *OptikonCentral) stopReadingServices() {
	close(oc.svcReadStopper)
}
