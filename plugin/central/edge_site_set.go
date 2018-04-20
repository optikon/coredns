package central

// EdgeSiteSet specifies the set of edges sites running a service.
type EdgeSiteSet map[EdgeSite]bool

// NewEdgeSiteSet returns a new instance of a set of edge sites.
func NewEdgeSiteSet() EdgeSiteSet {
	return make(EdgeSiteSet)
}

// Add adds a new entry to the set if it doesn't already exist.
func (ess EdgeSiteSet) Add(newEdgeSite EdgeSite) {
	if _, found := ess[newEdgeSite]; !found {
		ess[newEdgeSite] = true
	}
}

// Remove deletes any of the given instance in the set.
func (ess EdgeSiteSet) Remove(edgeSite EdgeSite) {
	if _, found := ess[edgeSite]; found {
		delete(ess, edgeSite)
	}
}

// ToSlice converts the set into a slice of edge sites.
func (ess EdgeSiteSet) ToSlice() []EdgeSite {
	result := make([]EdgeSite, len(ess))
	i := 0
	for edgeSite := range ess {
		result[i] = edgeSite
		i++
	}
	return result
}

// Size returns the size of the set.
func (ess EdgeSiteSet) Size() int {
	return len(ess)
}
