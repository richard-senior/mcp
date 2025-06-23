package util

import (
	"math"
)

// EllipticalArc represents an elliptical arc as defined in SVG A command
type EllipticalArc struct {
	Start    Point   // Start point
	End      Point   // End point
	RadiusX  float64 // Semi-major axis
	RadiusY  float64 // Semi-minor axis
	Rotation float64 // Rotation angle of the ellipse in radians
	Sweep    bool    // Sweep flag (false = clockwise, true = counterclockwise)
	LargeArc bool    // Large arc flag (false = small arc, true = large arc)

	// Computed values
	Sx, Sy float64 // Rotated center coordinates
	A0, A1 float64 // Start and end angles
	Da     float64 // Angle difference
	Ang    float64 // Rotation angle used in calculations
	Center Point   // Real center coordinates
}

// Constants
const (
	pi  = math.Pi
	pi2 = 2 * math.Pi

	// Accuracy constant for zero angle detection
	accZeroAng = 1e-10
)

// NewEllipticalArcFromGCode creates a new elliptical arc from GCode G02/G03 parameters
// G02/G03 arcs use I and J parameters to specify the relative offset from the current position to the center of the arc
// G02 is clockwise, G03 is counterclockwise
func NewEllipticalArcFromGCode(currentPos Point, endPos Point, i, j float64, clockwise bool) *EllipticalArc {
	// Calculate the center of the arc
	centerX := currentPos.X + i
	centerY := currentPos.Y + j
	center := Point{X: centerX, Y: centerY}

	// Calculate the radius (distance from current position to center)
	radiusX := math.Sqrt(i*i + j*j)
	radiusY := radiusX // In GRBL/GCode, arcs are always circular, so radiusX = radiusY

	// In GCode, G02 is clockwise and G03 is counterclockwise
	// In EllipticalArc, sweep=true means counterclockwise
	sweep := !clockwise

	// Determine if this is a large arc
	// This requires calculating the angle between start-center and end-center vectors
	startVecX := currentPos.X - centerX
	startVecY := currentPos.Y - centerY
	endVecX := endPos.X - centerX
	endVecY := endPos.Y - centerY

	// Calculate the angle between the two vectors using the dot product
	dotProduct := startVecX*endVecX + startVecY*endVecY
	startMag := math.Sqrt(startVecX*startVecX + startVecY*startVecY)
	endMag := math.Sqrt(endVecX*endVecX + endVecY*endVecY)

	// Avoid division by zero
	if startMag == 0 || endMag == 0 {
		return nil // Cannot create arc with zero radius
	}

	cosAngle := dotProduct / (startMag * endMag)
	// Clamp cosAngle to [-1, 1] to avoid NaN from floating point errors
	if cosAngle > 1.0 {
		cosAngle = 1.0
	} else if cosAngle < -1.0 {
		cosAngle = -1.0
	}

	angle := math.Acos(cosAngle)

	// Calculate the cross product to determine the direction
	crossProduct := startVecX*endVecY - startVecY*endVecX
	if crossProduct < 0 {
		angle = 2*math.Pi - angle
	}

	// In GCode, if the angle is > 180 degrees, we need to specify a full circle
	// For EllipticalArc, largeArc=true means angle > 180 degrees
	largeArc := false
	if (clockwise && angle < math.Pi) || (!clockwise && angle > math.Pi) {
		largeArc = true
	}

	// Create the elliptical arc
	arc := &EllipticalArc{
		Start:    currentPos,
		End:      endPos,
		RadiusX:  radiusX,
		RadiusY:  radiusY,
		Rotation: 0, // No rotation in standard GRBL/GCode arcs
		Sweep:    sweep,
		LargeArc: largeArc,
		Center:   center,
	}

	arc.Compute()
	return arc
}

// NewEllipticalArc creates a new elliptical arc with the given parameters
func NewEllipticalArc(start, end Point, radiusX, radiusY, rotation float64, sweep, largeArc bool) *EllipticalArc {
	arc := &EllipticalArc{
		Start:    start,
		End:      end,
		RadiusX:  radiusX,
		RadiusY:  radiusY,
		Rotation: rotation,
		Sweep:    sweep,
		LargeArc: largeArc,
	}
	arc.Compute()
	return arc
}

// NewEllipticalArcFromEllipse creates a new elliptical arc from an Ellipse struct
func NewEllipticalArcFromEllipse(ellipse Ellipse, start, end Point) *EllipticalArc {
	return NewEllipticalArc(
		start,
		end,
		ellipse.Radius1,
		ellipse.Radius2,
		ellipse.Angle,
		ellipse.SweepFlag,
		ellipse.LargeArcFlag,
	)
}

// GeneratePoints generates a series of points along the elliptical arc
func (arc *EllipticalArc) GeneratePoints(numPoints int) []Point {
	if numPoints < 2 {
		numPoints = 2
	}

	points := make([]Point, numPoints)

	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints-1)
		points[i] = arc.GetPoint(t)
	}

	return points
}

// GeneratePointsByDistance generates points with approximately the specified distance between them
func (arc *EllipticalArc) GeneratePointsByDistance(distance float64) []Point {
	if distance <= 0 {
		return []Point{arc.Start, arc.End}
	}

	// Estimate length and calculate number of points
	length := arc.GetLength(1.0, 1.0)
	numPoints := int(math.Ceil(length/distance)) + 1

	return arc.GeneratePoints(numPoints)
}

// ToLines converts the elliptical arc to a series of line segments
func (arc *EllipticalArc) ToLines(numSegments int) []Line {
	points := arc.GeneratePoints(numSegments + 1)
	lines := make([]Line, numSegments)

	for i := 0; i < numSegments; i++ {
		lines[i] = Line{
			Start: points[i],
			End:   points[i+1],
		}
	}

	return lines
}

// Reset resets the arc parameters to default values
func (arc *EllipticalArc) Reset() {
	arc.Start = Point{X: 0, Y: 0}
	arc.End = Point{X: 0, Y: 0}
	arc.RadiusX = 0
	arc.RadiusY = 0
	arc.Rotation = 0
	arc.Sweep = false
	arc.LargeArc = false
	arc.Compute()
}

// GetPoint calculates a point on the elliptical arc at parameter t (0 to 1)
func (arc *EllipticalArc) GetPoint(t float64) Point {
	// Calculate angle at parameter t
	angle := arc.A0 + (arc.Da * t)

	// Calculate point on the ellipse in rotated coordinates
	xx := arc.Sx + arc.RadiusX*math.Cos(angle)
	yy := arc.Sy + arc.RadiusY*math.Sin(angle)

	// Rotate back to original coordinate system
	c := math.Cos(-arc.Ang)
	s := math.Sin(-arc.Ang)
	x := xx*c - yy*s
	y := xx*s + yy*c

	return Point{X: x, Y: y}
}

// GetLength estimates the length of the arc
// mx, my are parameters for accuracy control (not fully implemented in this version)
func (arc *EllipticalArc) GetLength(mx, my float64) float64 {
	// This is a simplified implementation
	// For more accurate length calculation, numerical integration would be needed
	const segments = 100

	var length float64
	prevPoint := arc.GetPoint(0)

	for i := 1; i <= segments; i++ {
		t := float64(i) / segments
		point := arc.GetPoint(t)

		dx := point.X - prevPoint.X
		dy := point.Y - prevPoint.Y
		length += math.Sqrt(dx*dx + dy*dy)

		prevPoint = point
	}

	return length
}

// GetDeltaT calculates the parameter step size for a given arc length segment
func (arc *EllipticalArc) GetDeltaT(dl, mx, my float64) float64 {
	dt := dl / arc.GetLength(mx, my)
	n := math.Floor(1.0 / dt)
	if n < 1 {
		n = 1
	}
	return 1.0 / n
}

// Compute calculates all the internal parameters needed for the elliptical arc
func (arc *EllipticalArc) Compute() {
	// Set up rotation angle
	arc.Ang = pi - arc.Rotation
	sweep := arc.Sweep
	if arc.LargeArc {
		sweep = !sweep
	}

	// Calculate eccentricity
	e := arc.RadiusX / arc.RadiusY

	// Rotation matrix components
	c := math.Cos(arc.Ang)
	s := math.Sin(arc.Ang)

	// Transform start and end points
	ax := arc.Start.X*c - arc.Start.Y*s
	ay := arc.Start.X*s + arc.Start.Y*c
	bx := arc.End.X*c - arc.End.Y*s
	by := arc.End.X*s + arc.End.Y*c

	// Transform to circle
	ay *= e
	by *= e

	// Calculate midpoint between transformed points
	arc.Sx = 0.5 * (ax + bx)
	arc.Sy = 0.5 * (ay + by)

	// Calculate perpendicular vector
	vx := ay - by
	vy := bx - ax

	// Calculate distance from midpoint to center
	l := (arc.RadiusX*arc.RadiusX)/(vx*vx+vy*vy) - 0.25
	if l < 0 {
		l = 0
	}
	l = math.Sqrt(l)

	// Scale perpendicular vector
	vx *= l
	vy *= l

	// Determine center based on sweep flag
	if sweep {
		arc.Sx += vx
		arc.Sy += vy
	} else {
		arc.Sx -= vx
		arc.Sy -= vy
	}

	// Calculate start and end angles
	arc.A0 = atanXY(ax-arc.Sx, ay-arc.Sy)
	arc.A1 = atanXY(bx-arc.Sx, by-arc.Sy)

	// Transform back
	arc.Sy = arc.Sy / e

	// Calculate angle difference
	arc.Da = arc.A1 - arc.A0

	// Handle special cases
	if math.Abs(math.Abs(arc.Da)-pi) <= accZeroAng {
		// Half arc case
		db := (0.5 * (arc.A0 + arc.A1)) - atanXY(bx-ax, by-ay)

		// Normalize angle
		for db < -pi {
			db += pi2
		}
		for db > pi {
			db -= pi2
		}

		// Determine sweep direction
		newSweep := false
		if (db < 0.0) && (!arc.Sweep) {
			newSweep = true
		}
		if (db > 0.0) && arc.Sweep {
			newSweep = true
		}

		if newSweep {
			if arc.Da >= 0.0 {
				arc.A1 -= pi2
			}
			if arc.Da < 0.0 {
				arc.A0 -= pi2
			}
		}
	} else if arc.LargeArc {
		// Large arc case
		if arc.Da < pi && arc.Da >= 0.0 {
			arc.A1 -= pi2
		}
		if arc.Da > -pi && arc.Da < 0.0 {
			arc.A0 -= pi2
		}
	} else {
		// Small arc case
		if arc.Da > pi {
			arc.A1 -= pi2
		}
		if arc.Da < -pi {
			arc.A0 -= pi2
		}
	}

	// Recalculate angle difference
	arc.Da = arc.A1 - arc.A0

	// Calculate real center
	c = math.Cos(+arc.Ang)
	s = math.Sin(+arc.Ang)
	arc.Center = Point{
		X: arc.Sx*c - arc.Sy*s,
		Y: arc.Sx*s + arc.Sy*c,
	}
}

// ToLinesByDistance converts the elliptical arc to a series of line segments with approximately
// the specified distance between points
func (arc *EllipticalArc) ToLinesByDistance(distance float64) []Line {
	points := arc.GeneratePointsByDistance(distance)
	lines := make([]Line, len(points)-1)

	for i := 0; i < len(points)-1; i++ {
		lines[i] = Line{
			Start: points[i],
			End:   points[i+1],
		}
	}

	return lines
}

// ToEllipse converts the elliptical arc to an Ellipse struct
func (arc *EllipticalArc) ToEllipse() Ellipse {
	return Ellipse{
		Center1:      arc.Center,
		Radius1:      arc.RadiusX,
		Radius2:      arc.RadiusY,
		Angle:        arc.Rotation,
		LargeArcFlag: arc.LargeArc,
		SweepFlag:    arc.Sweep,
	}
}

// atanXY calculates the angle of a point (x,y) from the origin
// This is equivalent to atan2(y,x) but matches the original C++ implementation
func atanXY(x, y float64) float64 {
	return math.Atan2(y, x)
}
