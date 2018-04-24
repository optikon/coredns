package edge

import (
	"encoding/json"
	"sync"
)

// ConcurrentStringSet is a set of strings that can be safely shared between goroutines.
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

// Overwrite replaces all entries of the set with new ones.
func (cs *ConcurrentStringSet) Overwrite(newItems []string) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = make(map[string]bool)
	for _, item := range newItems {
		cs.items[item] = true
	}
}

// Contains checks whether or not an item is contained in the set.
func (cs *ConcurrentStringSet) Contains(item string) bool {
	cs.Lock()
	defer cs.Unlock()
	_, found := cs.items[item]
	return found
}

// ToJSON converts the current state of the set into JSON.
// TODO: Move this out of here.
func (cs *ConcurrentStringSet) ToJSON(meta EdgeSite) ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()
	itemList := make([]string, len(cs.items))
	i := 0
	for item := range cs.items {
		itemList[i] = item
		i++
	}
	update := TableUpdate{
		Meta:     meta,
		Services: itemList,
	}
	return json.Marshal(update)
}

// Size returns the number of elements in the set.
func (cs *ConcurrentStringSet) Size() int {
	cs.Lock()
	defer cs.Unlock()
	return len(cs.items)
}
