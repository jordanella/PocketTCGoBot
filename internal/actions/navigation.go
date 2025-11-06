package actions

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// Navigation actions - high-level screen transitions

// GoToShop navigates to the shop screen
func (l *Library) GoToShop() error {
	// Implementation stub
	return fmt.Errorf("not implemented")
}

// GoToSocial navigates to the social/friends screen
func (l *Library) GoToSocial() error {
	return fmt.Errorf("not implemented")
}

// GoToCollection navigates to the card collection
func (l *Library) GoToCollection() error {
	return fmt.Errorf("not implemented")
}

// GoToWonderPick navigates to Wonder Pick screen
func (l *Library) GoToWonderPick() error {
	return fmt.Errorf("not implemented")
}

// GoToMissions navigates to missions screen
// This is already declared in library.go, just providing implementation note
// func (l *Library) GoToMissions() error

// GoToMain navigates back to main screen from anywhere
func (l *Library) GoToMain(fromSocial bool) error {
	// Implementation based on AHK's GoToMain function
	return fmt.Errorf("not implemented")
}

// HomeAndMission is a complex navigation that goes home and optionally to missions
func (l *Library) HomeAndMission(homeOnly bool, completeSecondMission bool) error {
	// Based on AHK's HomeAndMission function
	// This is a critical function that handles:
	// - Level up detection and handling
	// - Navigation to shop then back to verify we're home
	// - Optional mission navigation
	// - Wonder pick detection for missions

	// Implementation stub
	return fmt.Errorf("not implemented")
}

// WaitForScreenLoad waits for a specific template to appear
func (l *Library) WaitForScreenLoad(tmpl cv.Template, timeout time.Duration) error {
	// Uses CV service to wait for template
	return fmt.Errorf("not implemented")
}

// HandlePopup attempts to close a popup using a template
func (l *Library) HandlePopup(tmpl cv.Template) error {
	return fmt.Errorf("not implemented")
}

// CloseAnyPopups tries to close common popups
func (l *Library) CloseAnyPopups() error {
	// Try clicking common popup close buttons
	commonPopups := []cv.Template{
		templates.Home, // Example
		// Add more common popup templates
	}

	for _, popup := range commonPopups {
		// Try to find and close each popup type
		_ = popup // TODO: implement
	}

	return nil
}
