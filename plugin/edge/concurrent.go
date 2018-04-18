package edge

import (
	"encoding/json"
	"sync"
)

// ConcurrentStringSet type that can be safely shared between goroutines.
type ConcurrentStringSet struct {
	sync.RWMutex
	items map[string]bool
}

// NewConcurrentStringSet creates a new concurrent set of strings.
func NewConcurrentStringSet() *ConcurrentStringSet {
	return &ConcurrentStringSet{
		items: make(map[string]bool),
	}
}

// Add adds an item to the concurrent set. Returns true if the value wasn't
// already in the set.
func (cs *ConcurrentStringSet) Add(item string) {
	cs.Lock()
	defer cs.Unlock()
	_, found := cs.items[item]
	cs.items[item] = true
	return !found
}

// Overwrite replaces all entries of the set with new ones.
func (cs *ConcurrentStringSet) Overwrite(newItems []string) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = make(map[string]bool)
	for _, newItem := range newItems {
		cs.items[newItem] = true
	}
}

// ToJSON converts the current state of the set into JSON bytes.
func (cs *ConcurrentStringSet) ToJSON() ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()
	keys := make([]string, len(cs.items))
	i := 0
	for k := range cs.items {
		keys[i] = k
		i++
	}
	return json.Marshal(keys)
}
