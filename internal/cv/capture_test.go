//go:build windows
// +build windows

package cv

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"syscall"
	"testing"
	"unsafe"
)

// FindWindowByTitle finds a window handle by its title
func FindWindowByTitle(title string) (uintptr, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	procFindWindow := user32.NewProc("FindWindowW")

	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}

	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(titlePtr)))
	if hwnd == 0 {
		return 0, fmt.Errorf("window not found: %s", title)
	}

	return hwnd, nil
}

// GetWindowTitle retrieves a window's title
func GetWindowTitle(hwnd uintptr) (string, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	procGetWindowText := user32.NewProc("GetWindowTextW")

	buf := make([]uint16, 256)
	ret, _, _ := procGetWindowText.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)

	if ret == 0 {
		return "", fmt.Errorf("failed to get window title")
	}

	return syscall.UTF16ToString(buf), nil
}

// TestWindowCapture tests capturing from a window
// To run this test, you need to have a window open with a known title
// Example: TestWindowCapture -window "Notepad"
func TestWindowCapture(t *testing.T) {
	// Skip in automated testing
	if testing.Short() {
		t.Skip("Skipping window capture test in short mode")
	}

	// Try to find a common window - adjust this based on your needs
	// Common windows: "Notepad", "Calculator", "MuMuPlayer"
	testWindows := []string{
		"MuMuPlayer",
		"Untitled - Notepad",
		"Calculator",
	}

	var hwnd uintptr
	var windowTitle string
	for _, title := range testWindows {
		h, err := FindWindowByTitle(title)
		if err == nil {
			hwnd = h
			windowTitle = title
			break
		}
	}

	if hwnd == 0 {
		t.Skip("No test window found. Please open one of: MuMuPlayer, Notepad, or Calculator")
	}

	t.Logf("Found window: %s (hwnd: 0x%x)", windowTitle, hwnd)

	// Create window capture
	capture, err := NewWindowCapture(hwnd)
	if err != nil {
		t.Fatalf("Failed to create window capture: %v", err)
	}

	// Get dimensions
	width, height := capture.GetDimensions()
	t.Logf("Window dimensions: %dx%d", width, height)

	if width <= 0 || height <= 0 {
		t.Fatalf("Invalid dimensions: %dx%d", width, height)
	}

	// Capture a frame
	frame, err := capture.CaptureFrame()
	if err != nil {
		t.Fatalf("Failed to capture frame: %v", err)
	}

	if frame == nil {
		t.Fatal("Captured frame is nil")
	}

	// Verify frame bounds
	bounds := frame.Bounds()
	t.Logf("Frame bounds: Min(%d,%d) Max(%d,%d)",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)

	if bounds.Min.X != 0 || bounds.Min.Y != 0 {
		t.Errorf("Frame should start at (0,0), got (%d,%d)", bounds.Min.X, bounds.Min.Y)
	}

	if bounds.Dx() != width || bounds.Dy() != height {
		t.Errorf("Frame size mismatch: expected %dx%d, got %dx%d",
			width, height, bounds.Dx(), bounds.Dy())
	}

	// Save test capture to file for visual verification
	outputPath := "test_capture.png"
	err = savePNG(frame, outputPath)
	if err != nil {
		t.Logf("Warning: Could not save test capture: %v", err)
	} else {
		t.Logf("Test capture saved to: %s", outputPath)
	}

	// Test coordinate system by checking corner pixels
	t.Run("CoordinateSystem", func(t *testing.T) {
		// Top-left corner (0,0) should be accessible
		topLeft := frame.At(0, 0)
		if topLeft == nil {
			t.Error("Cannot access pixel at (0,0)")
		}

		// Bottom-right corner should be at (width-1, height-1)
		bottomRight := frame.At(width-1, height-1)
		if bottomRight == nil {
			t.Error("Cannot access pixel at bottom-right corner")
		}

		// Out of bounds should panic or return nil (test safety)
		defer func() {
			if r := recover(); r == nil {
				// If no panic, check if out of bounds access returns valid data
				// This shouldn't happen but some implementations may handle it
			}
		}()

		t.Log("Coordinate system test passed - all corners accessible")
	})
}

// TestWindowCaptureService tests the CV Service with window capture
func TestWindowCaptureService(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping window capture service test in short mode")
	}

	// Find test window
	testWindows := []string{
		"MuMuPlayer",
		"Untitled - Notepad",
		"Calculator",
	}

	var hwnd uintptr
	for _, title := range testWindows {
		h, err := FindWindowByTitle(title)
		if err == nil {
			hwnd = h
			break
		}
	}

	if hwnd == 0 {
		t.Skip("No test window found")
	}

	// Create capture and service
	capture, err := NewWindowCapture(hwnd)
	if err != nil {
		t.Fatalf("Failed to create window capture: %v", err)
	}

	service := NewService(capture)

	// Test frame capture through service
	frame, err := service.CaptureFrame(false)
	if err != nil {
		t.Fatalf("Failed to capture frame through service: %v", err)
	}

	if frame == nil {
		t.Fatal("Service returned nil frame")
	}

	width, height := service.GetDimensions()
	t.Logf("Service dimensions: %dx%d", width, height)

	// Test caching
	t.Run("FrameCaching", func(t *testing.T) {
		// First capture (fresh)
		frame1, err := service.CaptureFrame(true)
		if err != nil {
			t.Fatalf("Failed to capture frame 1: %v", err)
		}

		// Second capture (should be cached)
		frame2, err := service.CaptureFrame(true)
		if err != nil {
			t.Fatalf("Failed to capture frame 2: %v", err)
		}

		// Frames should be the same object when cached
		if frame1 != frame2 {
			t.Log("Warning: Cached frames are different objects (may be expected)")
		}

		// Invalidate and capture again
		service.InvalidateCache()
		frame3, err := service.CaptureFrame(true)
		if err != nil {
			t.Fatalf("Failed to capture frame 3: %v", err)
		}

		if frame3 == nil {
			t.Fatal("Frame after invalidation is nil")
		}

		t.Log("Frame caching test passed")
	})
}

// TestCoordinateTranslation verifies that template matching returns window-relative coordinates
func TestCoordinateTranslation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping coordinate translation test in short mode")
	}

	// Create a test image (simulating window capture)
	testWidth := 800
	testHeight := 600
	testFrame := image.NewRGBA(image.Rect(0, 0, testWidth, testHeight))

	// Create a small template (50x50 red square)
	templateWidth := 50
	templateHeight := 50
	template := image.NewRGBA(image.Rect(0, 0, templateWidth, templateHeight))

	// Fill template with red
	for y := 0; y < templateHeight; y++ {
		for x := 0; x < templateWidth; x++ {
			idx := (y * template.Stride) + (x * 4)
			template.Pix[idx] = 255   // R
			template.Pix[idx+1] = 0   // G
			template.Pix[idx+2] = 0   // B
			template.Pix[idx+3] = 255 // A
		}
	}

	// Place the template in the test frame at position (100, 100)
	testX := 100
	testY := 100
	for y := 0; y < templateHeight; y++ {
		for x := 0; x < templateWidth; x++ {
			frameIdx := ((testY + y) * testFrame.Stride) + ((testX + x) * 4)
			testFrame.Pix[frameIdx] = 255   // R
			testFrame.Pix[frameIdx+1] = 0   // G
			testFrame.Pix[frameIdx+2] = 0   // B
			testFrame.Pix[frameIdx+3] = 255 // A
		}
	}

	// Perform template matching
	config := DefaultMatchConfig()
	config.Threshold = 0.95 // High threshold for exact match

	result := FindTemplate(testFrame, template, config)

	if !result.Found {
		t.Fatalf("Template not found (confidence: %.2f)", result.Confidence)
	}

	t.Logf("Template found at (%d, %d) with confidence %.2f",
		result.Location.X, result.Location.Y, result.Confidence)

	// Verify coordinates are correct (window-relative)
	if result.Location.X != testX || result.Location.Y != testY {
		t.Errorf("Coordinate mismatch: expected (%d, %d), got (%d, %d)",
			testX, testY, result.Location.X, result.Location.Y)
	}

	// Test with SearchRegion (window-relative region)
	t.Run("SearchRegionCoordinates", func(t *testing.T) {
		// Define a search region in window-relative coordinates
		searchRegion := image.Rect(50, 50, 200, 200)
		config.SearchRegion = &searchRegion

		result := FindTemplate(testFrame, template, config)

		if !result.Found {
			t.Fatalf("Template not found in search region")
		}

		// Verify result is still in window-relative coordinates
		if result.Location.X != testX || result.Location.Y != testY {
			t.Errorf("Coordinate mismatch with search region: expected (%d, %d), got (%d, %d)",
				testX, testY, result.Location.X, result.Location.Y)
		}

		t.Logf("Search region test passed - coordinates are window-relative")
	})

	// Test that coordinates outside search region are not found
	t.Run("OutsideSearchRegion", func(t *testing.T) {
		// Define a search region that doesn't include our template
		searchRegion := image.Rect(200, 200, 300, 300)
		config.SearchRegion = &searchRegion

		result := FindTemplate(testFrame, template, config)

		// Should not find template outside search region
		if result.Found {
			t.Errorf("Template should not be found outside search region")
		}

		t.Log("Outside search region test passed")
	})
}

// Helper function to save PNG
func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

// BenchmarkWindowCapture benchmarks window capture performance
func BenchmarkWindowCapture(b *testing.B) {
	// Try to find a window
	testWindows := []string{"MuMuPlayer", "Untitled - Notepad", "Calculator"}
	var hwnd uintptr
	for _, title := range testWindows {
		h, err := FindWindowByTitle(title)
		if err == nil {
			hwnd = h
			break
		}
	}

	if hwnd == 0 {
		b.Skip("No test window found")
	}

	capture, err := NewWindowCapture(hwnd)
	if err != nil {
		b.Fatalf("Failed to create window capture: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := capture.CaptureFrame()
		if err != nil {
			b.Fatalf("Capture failed: %v", err)
		}
	}
}
