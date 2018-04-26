package edge

import (
	"fmt"

	"github.com/mitchellh/hashstructure"
)

// Set is a set type for an interface{}.
type Set map[uint64]interface{}

// NewSet returns a new instance of a set.
func NewSet() Set {
	return make(Set)
}

// Add adds a new entry to the set if it doesn't already exist.
func (s Set) Add(value interface{}) {
	hash, err := hashstructure.Hash(value, nil)
	if err != nil {
		log.Errorf("type could not be hashed: %+v\n", value)
	}
	s[hash] = value
}

// Contains returns true if the given value is in the set.
func (s Set) Contains(value interface{}) bool {
	hash, err := hashstructure.Hash(value, nil)
	if err != nil {
		log.Errorf("type could not be hashed: %+v\n", value)
	}
	_, found := s[hash]
	return found
}

// Remove deletes the given value from the set.
func (s Set) Remove(value interface{}) {
	hash, err := hashstructure.Hash(value, nil)
	if err != nil {
		log.Errorf("type could not be hashed: %+v\n", value)
	}
	if _, exists := s[hash]; exists {
		delete(s, hash)
	}
}

// Len returns the number of elements in the set.
func (s Set) Len() int {
	return len(s)
}

// String returns a string representation of the set.
func (s Set) String() string {
	result := "{"
	for _, val := range s {
		result += fmt.Sprintf(" %+v", val)
	}
	if s.Len() == 0 {
		result += "}"
	} else {
		result += " }"
	}
	return result
}
