package cv

import (
	"fmt"
)

// CardDetector provides pack and card detection capabilities
type CardDetector struct {
	cv *Service
}

// NewCardDetector creates a new card detector
func NewCardDetector(cvService *Service) *CardDetector {
	return &CardDetector{
		cv: cvService,
	}
}

// BorderType represents different card border types for rarity detection
type BorderType string

const (
	BorderNormal   BorderType = "normal"
	Border1Star    BorderType = "1star"
	Border3Diamond BorderType = "3diamond"
	Border4Diamond BorderType = "4diamond" // Calculated, not detected directly
	BorderTrainer  BorderType = "trainer"
	BorderRainbow  BorderType = "rainbow"
	BorderFullArt  BorderType = "fullart"
	BorderLag      BorderType = "lag" // For checking if cards are still loading
)

// FindBorders detects cards of a specific border type in the current pack
func (cd *CardDetector) FindBorders(borderType BorderType) (int, error) {
	// This would use template matching to find cards with specific borders
	// Returns count of cards found with that border type
	// TODO: Implement actual detection logic
	return 0, fmt.Errorf("not implemented")
}

// DetectSixCardPack checks if the current pack is a 6-card pack
func (cd *CardDetector) DetectSixCardPack() (bool, error) {
	// 6-card packs have special indicators or layouts
	// TODO: Implement detection
	return false, nil
}

// DetectFourCardPack checks if the current pack is a 4-card pack
func (cd *CardDetector) DetectFourCardPack() (bool, error) {
	// 4-card packs have different layout
	// TODO: Implement detection
	return false, nil
}

// FindCard searches for a specific named card in the current pack
func (cd *CardDetector) FindCard(cardName string) (bool, error) {
	// Uses template matching for specific card detection
	// Used for things like Gimmighoul detection
	// TODO: Implement detection
	return false, nil
}

// PackValidation contains the results of pack validation
type PackValidation struct {
	IsValid      bool
	Reason       string
	StarCount    int
	CardCounts   map[BorderType]int
	TotalCards   int

	// Special card flags
	HasTrainer   bool
	HasFullArt   bool
	HasRainbow   bool
	HasCrown     bool
	HasImmersive bool
	HasShiny     bool

	// Pack type
	Is6Card      bool
	Is4Card      bool

	// S4T tradeables
	Found3Diamond int
	Found4Diamond int
	Found1Star    int
	FoundGimmighoul bool
}

// ValidatePack performs comprehensive pack validation based on config
func (cd *CardDetector) ValidatePack(config interface{}) (*PackValidation, error) {
	validation := &PackValidation{
		CardCounts: make(map[BorderType]int),
	}

	// Wait for pack to fully render
	for {
		lagCount, err := cd.FindBorders(BorderLag)
		if err != nil {
			return nil, err
		}
		if lagCount == 0 {
			break
		}
		// Sleep and retry
	}

	// Detect pack type
	is4Card, err := cd.DetectFourCardPack()
	if err != nil {
		return nil, err
	}
	validation.Is4Card = is4Card

	if !is4Card {
		is6Card, err := cd.DetectSixCardPack()
		if err != nil {
			return nil, err
		}
		validation.Is6Card = is6Card
	}

	// Determine total cards
	if validation.Is6Card {
		validation.TotalCards = 6
	} else if validation.Is4Card {
		validation.TotalCards = 4
	} else {
		validation.TotalCards = 5
	}

	// Count borders for each type
	borderTypes := []BorderType{
		BorderNormal,
		Border1Star,
		Border3Diamond,
		BorderTrainer,
		BorderRainbow,
		BorderFullArt,
	}

	for _, borderType := range borderTypes {
		count, err := cd.FindBorders(borderType)
		if err != nil {
			return nil, err
		}
		validation.CardCounts[borderType] = count
	}

	// Calculate 4-diamond count (by subtraction)
	normalCount := validation.CardCounts[BorderNormal]
	diamondCount3 := validation.CardCounts[Border3Diamond]
	starCount1 := validation.CardCounts[Border1Star]
	trainerCount := validation.CardCounts[BorderTrainer]
	rainbowCount := validation.CardCounts[BorderRainbow]
	fullArtCount := validation.CardCounts[BorderFullArt]

	diamond4Count := validation.TotalCards - normalCount - diamondCount3 - starCount1 - trainerCount - rainbowCount - fullArtCount
	if diamond4Count < 0 {
		diamond4Count = 0
	}
	validation.CardCounts[Border4Diamond] = diamond4Count

	// Set special card flags
	validation.HasTrainer = trainerCount > 0
	validation.HasFullArt = fullArtCount > 0
	validation.HasRainbow = rainbowCount > 0

	// Store counts for S4T
	validation.Found3Diamond = diamondCount3
	validation.Found4Diamond = diamond4Count
	validation.Found1Star = starCount1

	// TODO: Detect crown, immersive, shiny
	// TODO: Count stars (OCR or template-based)

	return validation, nil
}

// IsGodPack checks if a validated pack meets "god pack" criteria
func (cd *CardDetector) IsGodPack(validation *PackValidation, config interface{}) bool {
	// God pack logic from AHK:
	// - Check star count against minimum
	// - Check for required card types (trainer, full art, rainbow, etc.)
	// - Check pack-specific criteria

	// This will be implemented based on config settings
	// For now, return false
	return false
}

// CountStars counts the total stars in the current pack
func (cd *CardDetector) CountStars() (int, error) {
	// This would use OCR or template matching to count stars
	// TODO: Implement star counting
	return 0, nil
}

// DetectCrown checks if pack contains crown card
func (cd *CardDetector) DetectCrown() (bool, error) {
	// TODO: Implement crown detection
	return false, nil
}

// DetectImmersive checks if pack contains immersive card
func (cd *CardDetector) DetectImmersive() (bool, error) {
	// TODO: Implement immersive detection
	return false, nil
}

// DetectShiny checks if pack contains shiny card
func (cd *CardDetector) DetectShiny() (bool, error) {
	// TODO: Implement shiny detection
	return false, nil
}
