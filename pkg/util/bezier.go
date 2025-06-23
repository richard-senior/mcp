package util

import (
	"fmt"
	"math"
)

// An object which holds and manipulates information about Quadratic Bezier curves
type Bezier struct {
	Start   *Point
	End     *Point
	Control *Point
}

func NewQuadraticBezier(start, end, control *Point) (*Bezier, error) {
	if start == nil || end == nil || control == nil {
		return nil, fmt.Errorf("Must supply valid start end and control points for bezier constructor")
	}
	ret := &Bezier{
		Start:   start,
		End:     end,
		Control: control,
	}
	return ret, nil
}

// PointaliseByCount generates n points along a quadratic Bezier curve
// defined by start, control, and end points.
func (b *Bezier) PointaliseByCount(n int) *Path {
	if n < 2 {
		//here
		// Create an array with just the start and end points
		points := []*Point{
			b.Start,
			b.End,
		}

		// Create a new Path from these points
		path, err := NewPathFromPoints(points, "bezier_minimal")
		if err != nil {
			// If there's an error, return a default path
			return &Path{
				ID:     "bezier_error",
				Points: points,
			}
		}
		return path
	}

	points := make([]*Point, n)

	// Generate n points along the curve
	for i := 0; i < n; i++ {
		// Parameter t goes from 0 to 1
		t := float64(i) / float64(n-1)

		// Apply De Casteljau's algorithm for quadratic Bezier
		points[i] = quadraticBezierPoint(*b.Start, *b.Control, *b.End, t)
	}

	// Create a path from the generated points
	path, err := NewPathFromPoints(points, "bezier_curve")
	if err != nil {
		// If there's an error, return a default path
		return &Path{
			ID:     "bezier_error",
			Points: points,
		}
	}
	return path
}

// QuadraticBezierByDistance generates points along a quadratic Bezier curve
// with approximately maxDistance between consecutive points.
func QuadraticBezierByDistance(start, control, end Point, maxDistance float64) []*Point {
	if maxDistance <= 0 {
		return []*Point{
			NewPoint(start.X, start.Y),
			NewPoint(end.X, end.Y),
		}
	}

	// Estimate curve length to determine point count
	// This is an approximation - Bezier curves don't have a simple length formula
	estimatedLength := estimateCurveLength(start, control, end)
	pointCount := int(math.Ceil(estimatedLength/maxDistance)) + 1

	// Generate points with the estimated count
	points := make([]*Point, pointCount)

	for i := 0; i < pointCount; i++ {
		// Parameter t goes from 0 to 1
		t := float64(i) / float64(pointCount-1)
		points[i] = quadraticBezierPoint(start, control, end, t)
	}

	return points
}

// quadraticBezierPoint calculates a point on a quadratic Bezier curve at parameter t
func quadraticBezierPoint(start, control, end Point, t float64) *Point {
	// De Casteljau's algorithm for quadratic Bezier
	// B(t) = (1-t)²P₀ + 2(1-t)tP₁ + t²P₂

	mt := 1 - t
	mt2 := mt * mt
	t2 := t * t

	x := mt2*start.X + 2*mt*t*control.X + t2*end.X
	y := mt2*start.Y + 2*mt*t*control.Y + t2*end.Y

	// Use the NewPoint constructor to return a pointer
	return NewPoint(x, y)
}

// estimateCurveLength approximates the length of a quadratic Bezier curve
// using a simple polygon approximation with a reasonable number of segments
func estimateCurveLength(start, control, end Point) float64 {
	// Use a reasonable number of segments for length estimation
	const segments = 20

	var length float64
	prevPoint := start

	for i := 1; i <= segments; i++ {
		t := float64(i) / segments
		currentPoint := *quadraticBezierPoint(start, control, end, t)

		// Add distance between consecutive points
		dx := currentPoint.X - prevPoint.X
		dy := currentPoint.Y - prevPoint.Y
		length += math.Sqrt(dx*dx + dy*dy)

		prevPoint = currentPoint
	}

	return length
}
