package cv

import "image"

// Image region types
type Region struct {
	X1, Y1, X2, Y2 int
}

type Point struct {
	X, Y int
}

// Helper functions

// NewRegion creates a new region
func NewRegion(x1, y1, x2, y2 int) Region {
	return Region{X1: x1, Y1: y1, X2: x2, Y2: y2}
}

// Contains checks if a point is within the region
func (r Region) Contains(p Point) bool {
	return p.X >= r.X1 && p.X <= r.X2 && p.Y >= r.Y1 && p.Y <= r.Y2
}

// Width returns the width of the region
func (r Region) Width() int {
	return r.X2 - r.X1
}

// Height returns the height of the region
func (r Region) Height() int {
	return r.Y2 - r.Y1
}

// ToImageRectangle converts Region to *image.Rectangle for use with CV operations
func (r Region) ToImageRectangle() *image.Rectangle {
	return &image.Rectangle{
		Min: image.Point{X: r.X1, Y: r.Y1},
		Max: image.Point{X: r.X2, Y: r.Y2},
	}
}
