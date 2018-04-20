package edge

import (
	"encoding/json"
	"sync"

	"wwwin-github.cisco.com/edge/optikon-dns/plugin/central"
)

// ConcurrentStringSlice type that can be safely shared between goroutines.
type ConcurrentStringSlice struct {
	sync.RWMutex
	items []string
}

// NewConcurrentStringSlice creates a new concurrent slice of strings.
func NewConcurrentStringSlice() *ConcurrentStringSlice {
	return &ConcurrentStringSlice{
		items: make([]string, 0),
	}
}

// Overwrite replaces all entries of the slice with new ones.
func (cs *ConcurrentStringSlice) Overwrite(newItems []string) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = newItems
}

// ToJSON converts the current state of the slice into JSON bytes.
func (cs *ConcurrentStringSlice) ToJSON(meta central.EdgeSite) ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()
	update := central.TableUpdate{
		Meta:     meta,
		Services: cs.items,
	}
	return json.Marshal(update)
}
