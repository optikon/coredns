package edge

import "math"

const (
	earthRaidusKm = 6371 // Radius of the Earth in kilometers.
)

// degreesToRadians converts from degrees to radians.
func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}

// Distance calculates the shortest path between two coordinates on the surface
// of the Earth.
func Distance(lat1, lon1, lat2, lon2 float64) float64 {

	la1 := degreesToRadians(lat1)
	lo1 := degreesToRadians(lon1)
	la2 := degreesToRadians(lat2)
	lo2 := degreesToRadians(lon2)

	diffLat := la2 - la1
	diffLon := lo2 - lo1

	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(la1)*math.Cos(la2)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return c * earthRaidusKm
}
