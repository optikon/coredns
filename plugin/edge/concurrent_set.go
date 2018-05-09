package edge

import (
	"fmt"
	"sync"
)

// ConcurrentSet is a set of interface{} that can be safely shared between goroutines.
type ConcurrentSet struct {
	sync.RWMutex
	items Set
}

// NewConcurrentSet returns a new instance of a set.
func NewConcurrentSet() *ConcurrentSet {
	return &ConcurrentSet{
		items: make(Set),
	}
}

// Add adds a new entry to the set if it doesn't already exist.
func (cs *ConcurrentSet) Add(value interface{}) {
	cs.Lock()
	defer cs.Unlock()
	cs.items.Add(value)
}

// Contains returns true if the given value is in the set.
func (cs *ConcurrentSet) Contains(value interface{}) bool {
	cs.Lock()
	defer cs.Unlock()
	return cs.items.Contains(value)
}

// Remove deletes the given value from the set.
func (cs *ConcurrentSet) Remove(value interface{}) {
	cs.Lock()
	defer cs.Unlock()
	cs.items.Remove(value)
}

// Overwrite replaces the entire contents of the set with a new one.
func (cs *ConcurrentSet) Overwrite(newValues Set) {
	cs.Lock()
	defer cs.Unlock()
	cs.items = newValues
}

// Len returns the number of elements in the set.
func (cs *ConcurrentSet) Len() int {
	cs.Lock()
	defer cs.Unlock()
	return cs.items.Len()
}

// String returns the string representation of the set.
func (cs *ConcurrentSet) String() string {
	cs.Lock()
	defer cs.Unlock()
	return fmt.Sprintf("%+v", cs.items)
}
