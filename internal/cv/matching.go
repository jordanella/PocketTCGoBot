package cv

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// MatchResult contains template matching results
type MatchResult struct {
	Found      bool
	Location   image.Point
	Confidence float64
}

// MatchMethod defines template matching algorithm
type MatchMethod int

const (
	// MatchMethodSAD - Sum of Absolute Differences (fastest)
	MatchMethodSAD MatchMethod = iota
	// MatchMethodSSD - Sum of Squared Differences (balanced)
	MatchMethodSSD
	// MatchMethodNCC - Normalized Cross-Correlation (most accurate)
	MatchMethodNCC
)

// MatchConfig configures template matching
type MatchConfig struct {
	Method       MatchMethod
	Threshold    float64          // 0.0-1.0, higher = more strict
	SearchRegion *image.Rectangle // Optional: limit search area
	MaxMatches   int              // For FindAll, 0 = unlimited
}

// DefaultMatchConfig returns recommended settings
func DefaultMatchConfig() *MatchConfig {
	return &MatchConfig{
		Method:     MatchMethodSSD,
		Threshold:  0.85,
		MaxMatches: 1,
	}
}

// FindTemplate finds a template image within a larger image
func FindTemplate(haystack, needle *image.RGBA, config *MatchConfig) *MatchResult {
	if config == nil {
		config = DefaultMatchConfig()
	}

	haystackBounds := haystack.Bounds()
	needleBounds := needle.Bounds()

	needleWidth := needleBounds.Dx()
	needleHeight := needleBounds.Dy()

	// Validate dimensions
	if needleWidth > haystackBounds.Dx() || needleHeight > haystackBounds.Dy() {
		return &MatchResult{Found: false}
	}

	// Determine search area
	searchBounds := haystackBounds
	if config.SearchRegion != nil {
		searchBounds = config.SearchRegion.Intersect(haystackBounds)

		// Validate search region
		if searchBounds.Empty() {
			return &MatchResult{Found: false, Confidence: 0.0}
		}
	}

	bestScore := 0.0
	bestLocation := image.Point{}
	found := false

	// Scan search area
	// IMPORTANT: Use <= for Min, < for Max to ensure we don't go out of bounds
	maxY := searchBounds.Max.Y - needleHeight
	maxX := searchBounds.Max.X - needleWidth

	if maxY < searchBounds.Min.Y || maxX < searchBounds.Min.X {
		// Template doesn't fit in search region
		return &MatchResult{Found: false, Confidence: 0.0}
	}

	for y := searchBounds.Min.Y; y <= maxY; y++ {
		for x := searchBounds.Min.X; x <= maxX; x++ {
			// Calculate match score at this position
			score := calculateMatchScore(haystack, needle, x, y, config.Method)

			if score > bestScore {
				bestScore = score
				bestLocation = image.Point{x, y}
				if score >= config.Threshold {
					found = true
				}
			}
		}
	}

	return &MatchResult{
		Found:      found,
		Location:   bestLocation,
		Confidence: bestScore,
	}
}

// FindTemplateAll finds all matches above threshold
func FindTemplateAll(haystack, needle *image.RGBA, config *MatchConfig) []MatchResult {
	if config == nil {
		config = DefaultMatchConfig()
	}

	haystackBounds := haystack.Bounds()
	needleBounds := needle.Bounds()

	needleWidth := needleBounds.Dx()
	needleHeight := needleBounds.Dy()

	if needleWidth > haystackBounds.Dx() || needleHeight > haystackBounds.Dy() {
		return nil
	}

	searchBounds := haystackBounds
	if config.SearchRegion != nil {
		searchBounds = config.SearchRegion.Intersect(haystackBounds)

		// Validate search region
		if searchBounds.Empty() {
			return nil
		}
	}

	var results []MatchResult

	// Scan search area
	maxY := searchBounds.Max.Y - needleHeight
	maxX := searchBounds.Max.X - needleWidth

	if maxY < searchBounds.Min.Y || maxX < searchBounds.Min.X {
		// Template doesn't fit in search region
		return nil
	}

	for y := searchBounds.Min.Y; y <= maxY; y++ {
		for x := searchBounds.Min.X; x <= maxX; x++ {
			score := calculateMatchScore(haystack, needle, x, y, config.Method)

			if score >= config.Threshold {
				results = append(results, MatchResult{
					Found:      true,
					Location:   image.Point{x, y},
					Confidence: score,
				})

				if config.MaxMatches > 0 && len(results) >= config.MaxMatches {
					return results
				}
			}
		}
	}

	return results
}

// calculateMatchScore computes similarity between template and image region
func calculateMatchScore(haystack, needle *image.RGBA, x, y int, method MatchMethod) float64 {
	needleBounds := needle.Bounds()
	needleWidth := needleBounds.Dx()
	needleHeight := needleBounds.Dy()

	switch method {
	case MatchMethodSAD:
		return matchSAD(haystack, needle, x, y, needleWidth, needleHeight)
	case MatchMethodSSD:
		return matchSSD(haystack, needle, x, y, needleWidth, needleHeight)
	case MatchMethodNCC:
		return matchNCC(haystack, needle, x, y, needleWidth, needleHeight)
	default:
		return matchSSD(haystack, needle, x, y, needleWidth, needleHeight)
	}
}

// matchSAD - Sum of Absolute Differences (fastest, least accurate)
func matchSAD(haystack, needle *image.RGBA, x, y, width, height int) float64 {
	var sad uint64

	for ny := 0; ny < height; ny++ {
		for nx := 0; nx < width; nx++ {
			hIdx := ((y+ny)*haystack.Stride + (x+nx)*4)
			nIdx := (ny*needle.Stride + nx*4)

			// RGB difference
			sad += uint64(abs(int(haystack.Pix[hIdx]) - int(needle.Pix[nIdx])))
			sad += uint64(abs(int(haystack.Pix[hIdx+1]) - int(needle.Pix[nIdx+1])))
			sad += uint64(abs(int(haystack.Pix[hIdx+2]) - int(needle.Pix[nIdx+2])))
		}
	}

	// Normalize to 0-1 (lower SAD = better match)
	maxSAD := float64(width * height * 3 * 255)
	return 1.0 - (float64(sad) / maxSAD)
}

// matchSSD - Sum of Squared Differences (balanced)
func matchSSD(haystack, needle *image.RGBA, x, y, width, height int) float64 {
	var ssd uint64

	for ny := 0; ny < height; ny++ {
		for nx := 0; nx < width; nx++ {
			hIdx := ((y+ny)*haystack.Stride + (x+nx)*4)
			nIdx := (ny*needle.Stride + nx*4)

			// RGB squared difference
			dr := int(haystack.Pix[hIdx]) - int(needle.Pix[nIdx])
			dg := int(haystack.Pix[hIdx+1]) - int(needle.Pix[nIdx+1])
			db := int(haystack.Pix[hIdx+2]) - int(needle.Pix[nIdx+2])

			ssd += uint64(dr*dr + dg*dg + db*db)
		}
	}

	// Normalize to 0-1
	maxSSD := float64(width * height * 3 * 255 * 255)
	return 1.0 - (float64(ssd) / maxSSD)
}

// matchNCC - Normalized Cross-Correlation (slowest, most accurate)
func matchNCC(haystack, needle *image.RGBA, x, y, width, height int) float64 {
	var sumH, sumN, sumHN, sumHH, sumNN float64
	pixelCount := float64(width * height * 3)

	for ny := 0; ny < height; ny++ {
		for nx := 0; nx < width; nx++ {
			hIdx := ((y+ny)*haystack.Stride + (x+nx)*4)
			nIdx := (ny*needle.Stride + nx*4)

			// Process RGB channels
			for c := 0; c < 3; c++ {
				h := float64(haystack.Pix[hIdx+c])
				n := float64(needle.Pix[nIdx+c])

				sumH += h
				sumN += n
				sumHN += h * n
				sumHH += h * h
				sumNN += n * n
			}
		}
	}

	// Calculate means
	// meanH := sumH / pixelCount
	// meanN := sumN / pixelCount

	// Calculate numerator and denominator
	numerator := sumHN - (sumH * sumN / pixelCount)
	denomH := math.Sqrt(sumHH - (sumH * sumH / pixelCount))
	denomN := math.Sqrt(sumNN - (sumN * sumN / pixelCount))

	if denomH == 0 || denomN == 0 {
		return 0
	}

	// Correlation coefficient (-1 to 1, normalize to 0-1)
	correlation := numerator / (denomH * denomN)
	return (correlation + 1.0) / 2.0
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GrayscaleMatch performs grayscale template matching (faster)
func GrayscaleMatch(haystack, needle *image.RGBA, config *MatchConfig) *MatchResult {
	// Convert to grayscale for faster matching
	grayHaystack := toGrayscale(haystack)
	grayNeedle := toGrayscale(needle)

	return FindTemplate(grayHaystack, grayNeedle, config)
}

// toGrayscale converts RGBA to grayscale
func toGrayscale(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	gray := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			idx := (y * img.Stride) + (x * 4)
			r := img.Pix[idx]
			g := img.Pix[idx+1]
			b := img.Pix[idx+2]

			// Luminance formula
			grayValue := uint8((int(r)*299 + int(g)*587 + int(b)*114) / 1000)

			gray.Pix[idx] = grayValue
			gray.Pix[idx+1] = grayValue
			gray.Pix[idx+2] = grayValue
			gray.Pix[idx+3] = 255
		}
	}

	return gray
}

// ColorMatch performs color-based matching with tolerance
func ColorMatch(haystack *image.RGBA, targetColor color.RGBA, tolerance uint8) []image.Point {
	bounds := haystack.Bounds()
	var matches []image.Point

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			idx := (y * haystack.Stride) + (x * 4)
			r := haystack.Pix[idx]
			g := haystack.Pix[idx+1]
			b := haystack.Pix[idx+2]

			// Check if color matches within tolerance
			if colorDistance(r, g, b, targetColor.R, targetColor.G, targetColor.B) <= tolerance {
				matches = append(matches, image.Point{x, y})
			}
		}
	}

	return matches
}

func colorDistance(r1, g1, b1, r2, g2, b2 uint8) uint8 {
	dr := abs(int(r1) - int(r2))
	dg := abs(int(g1) - int(g2))
	db := abs(int(b1) - int(b2))
	return uint8((dr + dg + db) / 3)
}

// FindEdges simple edge detection
func FindEdges(img *image.RGBA, threshold uint8) *image.RGBA {
	bounds := img.Bounds()
	edges := image.NewRGBA(bounds)

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			// Sobel operator
			gx := getGradientX(img, x, y)
			gy := getGradientY(img, x, y)

			magnitude := uint8(math.Sqrt(float64(gx*gx + gy*gy)))

			idx := (y * edges.Stride) + (x * 4)
			if magnitude > threshold {
				edges.Pix[idx] = 255
				edges.Pix[idx+1] = 255
				edges.Pix[idx+2] = 255
			}
			edges.Pix[idx+3] = 255
		}
	}

	return edges
}

func getGradientX(img *image.RGBA, x, y int) int {
	// Sobel X kernel
	return getPixelIntensity(img, x+1, y-1) + 2*getPixelIntensity(img, x+1, y) + getPixelIntensity(img, x+1, y+1) -
		getPixelIntensity(img, x-1, y-1) - 2*getPixelIntensity(img, x-1, y) - getPixelIntensity(img, x-1, y+1)
}

func getGradientY(img *image.RGBA, x, y int) int {
	// Sobel Y kernel
	return getPixelIntensity(img, x-1, y+1) + 2*getPixelIntensity(img, x, y+1) + getPixelIntensity(img, x+1, y+1) -
		getPixelIntensity(img, x-1, y-1) - 2*getPixelIntensity(img, x, y-1) - getPixelIntensity(img, x+1, y-1)
}

func getPixelIntensity(img *image.RGBA, x, y int) int {
	idx := (y * img.Stride) + (x * 4)
	return int(img.Pix[idx])
}

// RegionAverage calculates average color in a region
func RegionAverage(img *image.RGBA, rect image.Rectangle) color.RGBA {
	var r, g, b uint64
	count := 0

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			idx := (y * img.Stride) + (x * 4)
			r += uint64(img.Pix[idx])
			g += uint64(img.Pix[idx+1])
			b += uint64(img.Pix[idx+2])
			count++
		}
	}

	if count == 0 {
		return color.RGBA{0, 0, 0, 255}
	}

	return color.RGBA{
		R: uint8(r / uint64(count)),
		G: uint8(g / uint64(count)),
		B: uint8(b / uint64(count)),
		A: 255,
	}
}

// CropRegion extracts a rectangular region from an image
func CropRegion(img *image.RGBA, rect image.Rectangle) *image.RGBA {
	cropped := image.NewRGBA(rect)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			c := img.RGBAAt(x, y)
			cropped.SetRGBA(x-rect.Min.X, y-rect.Min.Y, c)
		}
	}

	return cropped
}

// DebugMatch helps visualize match location
func DebugMatch(haystack *image.RGBA, result *MatchResult, needleSize image.Point) *image.RGBA {
	if !result.Found {
		return haystack
	}

	// Create copy
	debug := image.NewRGBA(haystack.Bounds())
	copy(debug.Pix, haystack.Pix)

	// Draw rectangle around match
	rect := image.Rect(
		result.Location.X,
		result.Location.Y,
		result.Location.X+needleSize.X,
		result.Location.Y+needleSize.Y,
	)

	drawRect(debug, rect, color.RGBA{255, 0, 0, 255})

	return debug
}

func drawRect(img *image.RGBA, rect image.Rectangle, col color.RGBA) {
	// Top and bottom
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.SetRGBA(x, rect.Min.Y, col)
		img.SetRGBA(x, rect.Max.Y-1, col)
	}
	// Left and right
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.SetRGBA(rect.Min.X, y, col)
		img.SetRGBA(rect.Max.X-1, y, col)
	}
}

// MatchStats provides matching statistics
type MatchStats struct {
	SearchArea   int     // Pixels searched
	Comparisons  int     // Template comparisons made
	Duration     float64 // Milliseconds
	BestScore    float64
	ThresholdMet bool
}

// FindTemplateWithStats returns match result and statistics
func FindTemplateWithStats(haystack, needle *image.RGBA, config *MatchConfig) (*MatchResult, *MatchStats) {
	// TODO: Implement with timing and stats
	result := FindTemplate(haystack, needle, config)
	stats := &MatchStats{
		BestScore:    result.Confidence,
		ThresholdMet: result.Found,
	}
	return result, stats
}

// ValidateMatch performs additional validation on match
func ValidateMatch(haystack, needle *image.RGBA, result *MatchResult, strictness float64) bool {
	if !result.Found {
		return false
	}

	// Re-check with higher threshold
	strictConfig := &MatchConfig{
		Method:    MatchMethodNCC,
		Threshold: strictness,
	}

	score := calculateMatchScore(
		haystack, needle,
		result.Location.X, result.Location.Y,
		strictConfig.Method,
	)

	return score >= strictness
}

// Error types
var (
	ErrTemplateTooLarge = fmt.Errorf("template larger than search image")
	ErrInvalidImage     = fmt.Errorf("invalid image provided")
)
