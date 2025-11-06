package cv

// CV operation options
type Option func(*cvOptions)

type cvOptions struct {
	threshold float64
	region    *Region
	variation int
}

// WithThreshold sets the matching threshold option
func WithThreshold(t float64) Option {
	return func(opts *cvOptions) {
		opts.threshold = t
	}
}

// WithRegion sets the search region option
func WithRegion(r *Region) Option {
	return func(opts *cvOptions) {
		opts.region = r
	}
}

// WithVariation sets the color variation tolerance option
func WithVariation(v int) Option {
	return func(opts *cvOptions) {
		opts.variation = v
	}
}
