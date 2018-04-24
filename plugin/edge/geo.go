package edge

import "math"

const (
	earthRaidusKm = 6371
	radianScalar  = math.Pi / 180.0
)

// Point represents a point on the planet.
type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// NewPoint returns a new Point for the given lon, lat coordinates.
func NewPoint(lat, lon float64) *Point {
	return &Point{
		Lat: lat,
		Lon: lon,
	}
}

// GreatCircleDistance calculates the shortest path between two coordinates
// on the surface of the Earth.
func (p1 *Point) GreatCircleDistance(p2 *Point) float64 {
	dLat := (p2.lat - p1.lat) * radianScalar
	dLon := (p2.lng - p1.lng) * radianScalar

	lat1 := p1.lat * radianScalar
	lat2 := p2.lat * radianScalar

	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)

	a := sinDLat*sinDLat + sinDLon*sinDLon*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRaidusKm * c
}
