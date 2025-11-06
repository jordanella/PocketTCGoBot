//go:build windows
// +build windows

package cv

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	procGetDC                  = user32.NewProc("GetDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procGetClientRect          = user32.NewProc("GetClientRect")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
)

const (
	SRCCOPY        = 0x00CC0020
	BI_RGB         = 0
	DIB_RGB_COLORS = 0
)

// RECT structure for Windows API
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// BITMAPINFOHEADER structure
type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// BITMAPINFO structure
type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

// WindowCapture handles direct window frame capture
type WindowCapture struct {
	hwnd   uintptr
	width  int
	height int
}

// NewWindowCapture creates a new window capture handler
func NewWindowCapture(hwnd uintptr) (*WindowCapture, error) {
	if hwnd == 0 {
		return nil, fmt.Errorf("invalid window handle")
	}

	// Get window dimensions
	var rect RECT
	ret, _, err := procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, fmt.Errorf("failed to get client rect: %v", err)
	}

	width := int(rect.Right - rect.Left)
	height := int(rect.Bottom - rect.Top)

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid window dimensions: %dx%d", width, height)
	}

	return &WindowCapture{
		hwnd:   hwnd,
		width:  width,
		height: height,
	}, nil
}

// CaptureFrame captures the current window frame as an image
func (wc *WindowCapture) CaptureFrame() (*image.RGBA, error) {
	// Get window DC
	hdcWindow, _, err := procGetDC.Call(wc.hwnd)
	if hdcWindow == 0 {
		return nil, fmt.Errorf("failed to get window DC: %v", err)
	}
	defer procReleaseDC.Call(wc.hwnd, hdcWindow)

	// Create compatible DC
	hdcMem, _, err := procCreateCompatibleDC.Call(hdcWindow)
	if hdcMem == 0 {
		return nil, fmt.Errorf("failed to create compatible DC: %v", err)
	}
	defer procDeleteDC.Call(hdcMem)

	// Create compatible bitmap
	hBitmap, _, err := procCreateCompatibleBitmap.Call(
		hdcWindow,
		uintptr(wc.width),
		uintptr(wc.height),
	)
	if hBitmap == 0 {
		return nil, fmt.Errorf("failed to create compatible bitmap: %v", err)
	}
	defer procDeleteObject.Call(hBitmap)

	// Select bitmap into memory DC
	_, _, _ = procSelectObject.Call(hdcMem, hBitmap)

	// Copy from window DC to memory DC
	ret, _, err := procBitBlt.Call(
		hdcMem,
		0, 0,
		uintptr(wc.width), uintptr(wc.height),
		hdcWindow,
		0, 0,
		SRCCOPY,
	)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt failed: %v", err)
	}

	// Prepare bitmap info
	var bi BITMAPINFO
	bi.BmiHeader.Size = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.Width = int32(wc.width)
	bi.BmiHeader.Height = -int32(wc.height) // Negative for top-down bitmap
	bi.BmiHeader.Planes = 1
	bi.BmiHeader.BitCount = 32
	bi.BmiHeader.Compression = BI_RGB

	// Allocate buffer for pixel data
	bufferSize := wc.width * wc.height * 4
	buffer := make([]byte, bufferSize)

	// Get bitmap bits
	ret, _, err = procGetDIBits.Call(
		hdcMem,
		hBitmap,
		0,
		uintptr(wc.height),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits failed: %v", err)
	}

	// Convert BGRA to RGBA
	img := image.NewRGBA(image.Rect(0, 0, wc.width, wc.height))
	for i := 0; i < len(buffer); i += 4 {
		// Windows uses BGRA, Go uses RGBA
		b := buffer[i]
		g := buffer[i+1]
		r := buffer[i+2]
		a := buffer[i+3]

		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}

	return img, nil
}

// GetDimensions returns the window dimensions
func (wc *WindowCapture) GetDimensions() (width, height int) {
	return wc.width, wc.height
}

// UpdateDimensions refreshes window dimensions (useful if window resizes)
func (wc *WindowCapture) UpdateDimensions() error {
	var rect RECT
	ret, _, err := procGetClientRect.Call(wc.hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return fmt.Errorf("failed to get client rect: %v", err)
	}

	wc.width = int(rect.Right - rect.Left)
	wc.height = int(rect.Bottom - rect.Top)

	if wc.width <= 0 || wc.height <= 0 {
		return fmt.Errorf("invalid window dimensions: %dx%d", wc.width, wc.height)
	}

	return nil
}
