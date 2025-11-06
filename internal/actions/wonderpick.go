package actions

import (
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

func (l *Library) DoWonderPickOnly() error {
	// Loop 1: Select a WonderPick and wait until we're in the card selection screen
	err := l.Action().
		UntilTemplateAppears(templates.Card,
			l.Action().
				// Click first wonderpick slot, then backup second slot
				Click(80, 390).
				Click(80, 460).
				Sleep(1*time.Second).
				// Check if out of WonderPick energy
				IfTemplateExists(templates.NoWPEnergy, l.Action().
					Sleep(2*time.Second).
					Click(137, 505). // Dismiss dialog
					Sleep(2*time.Second).
					Click(35, 515). // Go back
					Sleep(4*time.Second)).
				// TODO: Log "No WonderPick Energy left!"
				// Check if we're on WonderPick screen
				IfTemplateExists(templates.WonderPick, l.Action().
					// Look for and click the Button to start the pick
					FindAndClickCenter(templates.Button.InRegion(100, 367, 190, 480)).
					Sleep(3*time.Second)),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Loop 2: Click card until it's selected (Card template disappears)
	err = l.Action().
		Sleep(300*time.Millisecond).
		UntilTemplateDisappears(templates.Card,
			l.Action().
				Click(183, 350). // Click card position
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Loop 3: Click through results until Skip or WonderPick screen appears
	err = l.Action().
		UntilAnyTemplate([]cv.Template{templates.Skip, templates.WonderPick},
			l.Action().
				Click(146, 494). // Click to advance
				Sleep(1*time.Second).
				IfTemplateExistsClick(templates.Card), // If Card still showing, click it
			45).
		Execute()
	if err != nil {
		return err
	}

	// Loop 4: Navigate back to Shop (main menu)
	err = l.Action().
		UntilTemplateAppears(templates.Shop,
			l.Action().
				Sleep(1*time.Second).
				IfTemplateExistsClick(templates.Skip).
				IfTemplateExistsClick(templates.Card).
				Do(func() error { // Otherwise send ESC key to go back
					// Only send ESC if neither Skip nor Card are visible
					if !l.Action().TemplateExists(templates.Skip) &&
						!l.Action().TemplateExists(templates.Card) {
						return l.bot.ADB().SendKey("ESCAPE")
					}
					return nil
				}).
				Sleep(4*time.Second),
			45).
		Execute()

	return err
}

// DoWonderPick performs the complete WonderPick routine
// This is a clean translation of the AutoHotkey DoWonderPick() function
func (l *Library) DoWonderPick() error {

	err := l.Action().
		UntilTemplateAppearsClickPoint(templates.Shop, 40, 515, 0). // Navigate to main menu by clicking until Shop icon appears
		UntilTemplateAppearsClickPoint(templates.WonderPick, 59, 429, 0).
		Execute()
	if err != nil {
		return err
	}

	// Do the actual wonder picks
	err = l.DoWonderPickOnly()
	if err != nil {
		return err
	}

	// Navigate back to missions screen
	// Keep clicking until we see Missions, DexMissions, or DailyMissions icons
	missionTemplates := []cv.Template{
		templates.Missions,
		templates.DexMissions,
		templates.DailyMissions,
	}

	err = l.Action().
		UntilAnyTemplate(missionTemplates, l.Action().
			Click(261, 478).
			Sleep(1*time.Second), 0).
		FindAndClickCenter(templates.FirstMission).
		Sleep(1 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	// Collect mission rewards loop
	// Click through rewards until we're back at the main menu (Shop icon visible)
	err = l.Action().
		UntilTemplateAppears(templates.Shop,
			l.Action().
				Click(139, 424).
				Sleep(1*time.Second).
				IfTemplateExistsClickPoint(templates.Button.InRegion(145, 447, 258, 480), 110, 369). // If we see a Button template, click the dismiss area
				IfTemplateExistsClickPoint(templates.Shop, 139, 492),
			45). // Max 45 attempts
		Execute()

	return err
}
