package edge

import (
	"fmt"
	"sync"
)

// ServiceDNS is the DNS name of a Kubernetes service.
type ServiceDNS string

// ServiceTable specifies the mapping from service DNS names to edge sites.
type ServiceTable map[ServiceDNS]EdgeSiteSet

// ServiceTableUpdate encapsulates all the information sent in a table update
// from an edge site.
type ServiceTableUpdate struct {
	Meta     EdgeSite     `json:"meta"`
	Services []ServiceDNS `json:"services"`
}

// ConcurrentServiceTable is a table that can be safely shared between goroutines.
type ConcurrentServiceTable struct {
	sync.RWMutex
	table ServiceTable
}

// NewConcurrentServiceTable creates a new concurrent table.
func NewConcurrentServiceTable() *ConcurrentServiceTable {
	return &ConcurrentTable{
		table: make(Table),
	}
}

// Lookup performs a locked lookup for edge sites running a particular service.
func (cst *ConcurrentServiceTable) Lookup(svc ServiceDNS) (EdgeSiteSet, bool) {
	cst.Lock()
	defer cst.Unlock()
	return cst.table[svc]
}

// Update adds new entries to the table.
func (cst *ConcurrentServiceTable) Update(ip string, lon, lat float64, serviceNames []ServiceDNS) {

	// Print a log message.
	log.Infof("==========\nUpdating Table (IP: %s, Lon: %f, Lat: %f) with services: %+v (len: %d)\n==========\n", ip, lon, lat, serviceDomains, len(serviceDomains))

	// Create a struct to represent the edge site.
	myEdgeSite := EdgeSite{
		IP:  ip,
		Lon: lon,
		Lat: lat,
	}

	// Lock down the table.
	cst.Lock()
	defer cst.Unlock()

	// Loop over services and add the new entries.
	serviceNameSet := make(map[ServiceDNS]bool)
	for _, serviceDomain := range serviceDomains {
		serviceDomainSet[serviceDomain] = true
		if edgeSites, found := cst.table[serviceDomain]; found {
			edgeSites.Add(myEdgeSite)
		} else {
			newSet := NewEdgeSiteSet()
			newSet.Add(myEdgeSite)
			cst.table[serviceDomain] = newSet
		}
	}

	// Loop over the existing services and remove any that are no longer running.
	// NOTE: We need to remove empty entries _after_ iterating over the map.
	entriesToDelete := make([]string, 0)
	for serviceDomain, edgeSiteSet := range cst.table {
		if serviceDomainSet[serviceDomain] {
			continue
		}
		edgeSiteSet.Remove(myEdgeSite)
		if edgeSiteSet.Size() == 0 {
			entriesToDelete = append(entriesToDelete, serviceDomain)
		}
	}

	// Delete empty entries.
	for _, entry := range entriesToDelete {
		delete(cst.table, entry)
	}

	// Print the updated table.
	fmt.Printf("----------\nUpdated Table: %+v\n----------\n", cst.table)

}
