package util

// Represents a straight line from the start point to the end point
type Line struct {
	Start, End Point
}

// A structure for holding information about circular Arc's
type Arc struct {
	Start, End Point
	Center     Point
	Radius     float64
}

// A Structure for holding information about elliptical curves
type Ellipse struct {
	Center1      Point
	Radius1      float64
	Radius2      float64
	Angle        float64
	LargeArcFlag bool
	SweepFlag    bool
}

// A structure for holding information about Quadratic Bezier Curves
type QuadraticBezier struct {
	Start, End Point
	Control    Point
}
