package edge

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Starts the process of reading Kubernetes services every read interval.
func (e *Edge) startReadingServices() {
	ticker := time.NewTicker(e.svcReadInterval)
	e.svcReadChan = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				services, err := e.clientset.CoreV1().Services("").List(metaV1.ListOptions{})
				if err != nil {
					log.Errorf("couldn't read locally running Kubernetes services: %v", err)
					continue
				}
				serviceSet := make(Set)
				for _, service := range services.Items {
					serviceSet.Add(generateServiceDNS(&service))
				}
				e.services.Overwrite(serviceSet)
				log.Infof("Updated services: %+v", e.services)
				e.table.Update(e.ip, e.geoCoords, serviceSet)
			case <-e.svcReadChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Generates a services DNS that looks like my-svc.my-namespace.svc.cluster.external
func generateServiceDNS(svc *v1.Service) ServiceDNS {
	return ServiceDNS(fmt.Sprintf("%s.%s.svc.cluster.external", svc.GetName(), svc.GetNamespace()))
}

// Stops reading local Kubernetes services.
func (e *Edge) stopReadingServices() {
	close(e.svcReadChan)
}
