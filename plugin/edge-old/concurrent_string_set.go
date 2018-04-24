package edge

import (
	"encoding/json"
	"fmt"
	"sync"

	"wwwin-github.cisco.com/edge/optikon-dns/plugin/central"
)

// ConcurrentStringSet type that can be safely shared between goroutines.
type ConcurrentStringSet struct {
	sync.RWMutex
	items map[string]bool
}

// NewConcurrentStringSet creates a new concurrent slice of strings.
func NewConcurrentStringSet() *ConcurrentStringSet {
	return &ConcurrentStringSet{
		items: make(map[string]bool),
	}
}

// Overwrite replaces all entries of the slice with new ones.
func (cs *ConcurrentStringSet) Overwrite(newItems []string) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = make(map[string]bool)
	for _, item := range newItems {
		cs.items[item] = true
	}
	fmt.Printf("==========\nUpdated service list: %+v\n==========\n", cs.items)
}

// Contains check whether or not a service is contained in the set.
func (cs *ConcurrentStringSet) Contains(service string) bool {
	cs.Lock()
	defer cs.Unlock()
	_, found := cs.items[service]
	return found
}

// ToJSON converts the current state of the slice into JSON bytes.
func (cs *ConcurrentStringSet) ToJSON(meta central.EdgeSite) ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()
	serviceList := make([]string, len(cs.items))
	i := 0
	for service := range cs.items {
		serviceList[i] = service
		i++
	}
	update := central.TableUpdate{
		Meta:     meta,
		Services: serviceList,
	}
	return json.Marshal(update)
}

// Size returns the number of elements in the slice.
func (cs *ConcurrentStringSet) Size() int {
	cs.Lock()
	defer cs.Unlock()
	return len(cs.items)
}
