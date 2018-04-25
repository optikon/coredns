package edge

import (
	"fmt"

	"github.com/miekg/dns"
)

var locDomain = fmt.Sprintf("%s.site.", pluginName)

// Parses and removes the LOC record from the Extra fields of a DNS message.
func extractLocationRecord(r *dns.Msg) (*Point, bool) {
	// TODO: FINISH
	return &Point{}, false
}

// Inserts a new LOC record into a DNS request under the Extra fields.
func insertLocationRecord(r *dns.Msg, geoCoords *Point) {
	loc := new(dns.LOC)
	fmt.Println(loc)
	// TODO: FINISH
	return
}
