package cv

// Template type
type Template struct {
	Name      string
	Path      string
	Threshold float64
	Region    *Region
	Scale     float64
}

// Builder methods

// InRegion sets the search region for the template
func (t Template) InRegion(x1, y1, x2, y2 int) Template {
	region := NewRegion(x1, y1, x2, y2)
	t.Region = &region
	return t
}

// WithThreshold sets the matching threshold
func (t Template) WithThreshold(threshold float64) Template {
	t.Threshold = threshold
	return t
}

// WithScale sets the scale factor
func (t Template) WithScale(scale float64) Template {
	t.Scale = scale
	return t
}
