package central

import (
	"fmt"
	"sync"
)

// Table specifies the mapping from service DNS names to edge sites.
type Table map[string]EdgeSiteSet

// TableUpdate encapsulates all the information send in a table update
// from an edge site.
type TableUpdate struct {
	Meta     EdgeSite `json:"meta"`
	Services []string `json:"services"`
}

// ConcurrentTable type that can be safely shared between goroutines.
type ConcurrentTable struct {
	sync.RWMutex
	items Table
}

// NewConcurrentTable creates a new concurrent table.
func NewConcurrentTable() *ConcurrentTable {
	return &ConcurrentTable{
		items: make(Table),
	}
}

// Lookup performs a locked lookup for edge sites.
func (ct *ConcurrentTable) Lookup(key string) ([]EdgeSite, bool) {
	ct.Lock()
	defer ct.Unlock()
	set, found := ct.items[key]
	return set.ToSlice(), found
}

// Update adds new entries to the table.
func (ct *ConcurrentTable) Update(ip string, lon, lat float64, serviceDomains []string) {

	// Print a log message.
	fmt.Printf("Updating Table (IP: %s, Lon: %f, Lat: %f) with services: %+v (len: %d)\n", ip, lon, lat, serviceDomains, len(serviceDomains))

	// Create a struct to represent the edge site.
	myEdgeSite := EdgeSite{
		IP:  ip,
		Lon: lon,
		Lat: lat,
	}

	// Lock down the table.
	ct.Lock()
	defer ct.Unlock()

	// Loop over services and add the new entries.
	serviceDomainSet := make(map[string]bool)
	for _, serviceDomain := range serviceDomains {
		serviceDomainSet[serviceDomain] = true
		if edgeSites, found := ct.items[serviceDomain]; found {
			edgeSites.Add(myEdgeSite)
		} else {
			newSet := NewEdgeSiteSet()
			newSet.Add(myEdgeSite)
			ct.items[serviceDomain] = newSet
		}
	}

	// Loop over the existing services and remove and that are no longer running.
	entriesToDelete := make([]string, 0)
	for serviceDomain, edgeSiteSet := range ct.items {
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
		delete(ct.items, entry)
	}

	// Print the updated table.
	fmt.Printf("Updated Table: %+v\n", ct.items)

}
