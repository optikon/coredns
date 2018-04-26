package edge

import "fmt"

// This represents an existing value in a Golang set.
// NOTE: struct{} better than bool b/c struct{} is 0 bytes.
var exists = struct{}{}

// Set is a set type for an interface{}.
type Set map[interface{}]struct{}

// NewSet returns a new instance of a set.
func NewSet() Set {
	return make(Set)
}

// Add adds a new entry to the set if it doesn't already exist.
func (s Set) Add(value interface{}) {
	s[value] = exists
}

// Contains returns true if the given value is in the set.
func (s Set) Contains(value interface{}) bool {
	_, found := s[value]
	return found
}

// Remove deletes the given value from the set.
func (s Set) Remove(value interface{}) {
	if _, exists := s[value]; exists {
		delete(s, value)
	}
}

// Len returns the number of elements in the set.
func (s Set) Len() int {
	return len(s)
}

// String returns a string representation of the set.
func (s Set) String() string {
	result := "{"
	for val := range s {
		result += fmt.Sprintf(" %+v", val)
	}
	if s.Len() == 0 {
		result += "}"
	} else {
		result += " }"
	}
	return result
}
