package game

import "math/rand"

// MapObject represents a placed object on the map.
type MapObject struct {
	Type string  `json:"type"` // "tree", "lake", "jail"
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// Object sizes (must match Godot client scene sizes).
var objectSizes = map[string][2]float64{
	"jail": {200, 200},
	"lake": {400, 300},
	"tree": {80, 120},
}

// GenerateMapObjects creates randomized map object placements.
// All clients must use these positions to see the same map.
func GenerateMapObjects() []MapObject {
	var objects []MapObject
	var placed []placedObj

	mapCenterX := float64(MapWidth) / 2
	mapCenterY := float64(MapHeight) / 2

	// Place jails first, then lakes, then trees (same order as client)
	for i := 0; i < JailCount; i++ {
		if obj, ok := placeObject("jail", placed, mapCenterX, mapCenterY); ok {
			objects = append(objects, obj)
			placed = append(placed, placedObj{x: obj.X, y: obj.Y})
		}
	}
	for i := 0; i < LakeCount; i++ {
		if obj, ok := placeObject("lake", placed, mapCenterX, mapCenterY); ok {
			objects = append(objects, obj)
			placed = append(placed, placedObj{x: obj.X, y: obj.Y})
		}
	}
	for i := 0; i < TreeCount; i++ {
		if obj, ok := placeObject("tree", placed, mapCenterX, mapCenterY); ok {
			objects = append(objects, obj)
			placed = append(placed, placedObj{x: obj.X, y: obj.Y})
		}
	}

	return objects
}

type placedObj struct {
	x, y float64
}

func placeObject(objType string, placed []placedObj, centerX, centerY float64) (MapObject, bool) {
	size := objectSizes[objType]
	marginX := size[0] / 2
	marginY := size[1] / 2

	const maxAttempts = 100
	for i := 0; i < maxAttempts; i++ {
		x := marginX + rand.Float64()*(float64(MapWidth)-2*marginX)
		y := marginY + rand.Float64()*(float64(MapHeight)-2*marginY)

		// Exclude area around map center
		if Distance(x, y, centerX, centerY) < ObjectSpawnRadius+size[0]/2 {
			continue
		}

		// Check distance from other placed objects
		tooClose := false
		for _, p := range placed {
			if Distance(x, y, p.x, p.y) < ObjectMinDistance {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}

		return MapObject{Type: objType, X: x, Y: y}, true
	}

	// Fallback: place anyway
	x := marginX + rand.Float64()*(float64(MapWidth)-2*marginX)
	y := marginY + rand.Float64()*(float64(MapHeight)-2*marginY)
	return MapObject{Type: objType, X: x, Y: y}, true
}
