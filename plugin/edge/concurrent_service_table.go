package edge

import (
	"net"
	"sync"
)

// ServiceTable specifies the mapping from service DNS names to edge sites.
type ServiceTable map[string]Set

// ServiceTableUpdate encapsulates all the information sent in a table update
// from an edge site.
type ServiceTableUpdate struct {
	Meta     Site `json:"meta"`
	Services Set  `json:"services"`
}

// ConcurrentServiceTable is a table that can be safely shared between goroutines.
type ConcurrentServiceTable struct {
	sync.RWMutex
	table ServiceTable
}

// NewConcurrentServiceTable creates a new concurrent table.
func NewConcurrentServiceTable() *ConcurrentServiceTable {
	return &ConcurrentServiceTable{
		table: make(ServiceTable),
	}
}

// Lookup performs a locked lookup for edge sites running a particular service.
func (cst *ConcurrentServiceTable) Lookup(svc string) (Set, bool) {
	cst.Lock()
	defer cst.Unlock()
	set, found := cst.table[svc]
	return set, found
}

// Update adds new entries to the table.
func (cst *ConcurrentServiceTable) Update(ip net.IP, geoCoords Point, serviceNames Set) {

	// Create a struct to represent the edge site.
	mySite := Site{
		IP:        ip,
		GeoCoords: geoCoords,
	}

	// Lock down the table.
	cst.Lock()
	defer cst.Unlock()

	// Loop over services and add the new entries.
	for _, val := range serviceNames {
		serviceName := val.(string)
		if edgeSites, found := cst.table[serviceName]; found {
			edgeSites.Add(mySite)
		} else {
			newSet := NewSet()
			newSet.Add(mySite)
			cst.table[serviceName] = newSet
		}
	}

	// Loop over the existing services and remove any that are no longer running.
	// NOTE: We need to remove empty entries *after* iterating over the map.
	entriesToDelete := make([]string, 0)
	for serviceName, edgeSiteSet := range cst.table {
		if serviceNames.Contains(serviceName) {
			continue
		}
		edgeSiteSet.Remove(mySite)
		if edgeSiteSet.Len() == 0 {
			entriesToDelete = append(entriesToDelete, serviceName)
		}
	}

	// Delete empty entries.
	for _, entry := range entriesToDelete {
		delete(cst.table, entry)
	}

	// Log the new table.
	log.Infof("Updated table: %+v", cst.table)
}
