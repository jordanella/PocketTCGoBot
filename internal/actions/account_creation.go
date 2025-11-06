package actions

import (
	"time"

	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// DoTutorial performs the complete account creation and tutorial flow
func (l *Library) DoTutorial() error {
	// Phase 1: Birth date entry
	if err := l.enterBirthDate(); err != nil {
		return err
	}

	// Phase 2: Accept Terms of Service and Privacy Policy
	if err := l.acceptTermsOfService(); err != nil {
		return err
	}

	// Phase 3: Create save data and handle account linking
	if err := l.createSaveData(); err != nil {
		return err
	}

	// Phase 4: Character customization (name input)
	if err := l.setupCharacterName(); err != nil {
		return err
	}

	// Phase 5: First pack opening tutorial
	if err := l.tutorialFirstPack(); err != nil {
		return err
	}

	// Phase 6: Mission registration
	if err := l.tutorialMissionRegistration(); err != nil {
		return err
	}

	// Phase 7: Second pack opening tutorial
	if err := l.tutorialSecondPack(); err != nil {
		return err
	}

	// Phase 8: Wonder Pick tutorial
	if err := l.tutorialWonderPick(); err != nil {
		return err
	}

	// Navigate to main menu to complete tutorial
	err := l.Action().
		UntilTemplateAppears(templates.Main,
			l.Action().
				Click(192, 449).
				Sleep(1*time.Second),
			45).
		Execute()

	return err
}

// enterBirthDate handles the birth date entry screen (Country, Month, Year)
func (l *Library) enterBirthDate() error {
	// Click on Country field to start
	err := l.Action().
		FindAndClickCenter(templates.Country).
		Sleep(3*time.Second).
		Click(143, 370).
		Sleep(3 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	// Select Month
	err = l.Action().
		Click(80, 400).
		Sleep(3*time.Second).
		Click(80, 375).
		Sleep(3*time.Second).
		UntilTemplateDisappears(templates.Month,
			l.Action().
				Sleep(3*time.Second).
				Click(142, 159).
				Sleep(3*time.Second).
				Click(80, 400).
				Sleep(3*time.Second).
				Click(80, 375).
				Sleep(3*time.Second).
				Click(82, 422).
				Sleep(3*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Select Year
	err = l.Action().
		Click(200, 400).
		Sleep(3*time.Second).
		Click(200, 375).
		Sleep(3*time.Second).
		UntilTemplateDisappears(templates.Year,
			l.Action().
				Sleep(3*time.Second).
				Click(142, 159).
				Sleep(3*time.Second).
				Click(142, 159).
				Sleep(3*time.Second).
				Click(200, 400).
				Sleep(3*time.Second).
				Click(200, 375).
				Sleep(3*time.Second).
				Click(142, 159).
				Sleep(3*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Handle Country selection if needed
	err = l.Action().
		Sleep(3*time.Second).
		IfTemplateExists(templates.CountrySelect,
			l.Action().
				FindAndClickCenter(templates.CountrySelect2).
				Sleep(500*time.Millisecond).
				UntilTemplateAppears(templates.Birth,
					l.Action().
						Sleep(3*time.Second).
						IfTemplateExistsClickPoint(templates.CountrySelect, 124, 250).
						Do(func() error {
							// Click confirm if CountrySelect not visible and Birth not found
							if !l.Action().TemplateExists(templates.CountrySelect) &&
								!l.Action().TemplateExists(templates.Birth) {
								return l.bot.ADB().Click(140, 474)
							}
							return nil
						}),
					45),
		).
		Execute()
	if err != nil {
		return err
	}

	// Click birth confirmation if CountrySelect was not shown
	return l.Action().IfTemplateExistsClickPoint(templates.CountrySelect, 140, 474).Execute()
}

// acceptTermsOfService handles TOS and Privacy Policy acceptance
func (l *Library) acceptTermsOfService() error {
	// Wait for TOS screen and confirm birth
	err := l.Action().
		UntilTemplateAppears(templates.TosScreen,
			l.Action().
				Click(203, 371).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Open TOS then dismiss it
	err = l.Action().
		UntilTemplateAppears(templates.Tos,
			l.Action().
				Click(139, 299).
				Sleep(1*time.Second),
			45).
		UntilTemplateAppears(templates.TosScreen,
			l.Action().
				Click(142, 486).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Open Privacy Policy then dismiss it
	err = l.Action().
		UntilTemplateAppears(templates.Privacy,
			l.Action().
				Click(142, 339).
				Sleep(1*time.Second),
			45).
		UntilTemplateAppears(templates.TosScreen,
			l.Action().
				Click(142, 486).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Accept both TOS and Privacy
	err = l.Action().
		Sleep(750*time.Millisecond).
		Click(261, 374).
		Sleep(750*time.Millisecond).
		Click(261, 406).
		Sleep(750*time.Millisecond).
		Click(145, 484).
		Sleep(750 * time.Millisecond).
		Execute()

	return err
}

// createSaveData handles save data creation and account linking
func (l *Library) createSaveData() error {
	// Wait for Save screen and click create
	err := l.Action().
		UntilTemplateAppears(templates.Save,
			l.Action().
				Click(145, 484).
				Sleep(1*time.Second).
				Click(261, 406).
				Sleep(1*time.Second).
				Click(261, 374).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Confirm save creation
	err = l.Action().
		Sleep(1*time.Second).
		Click(143, 348).
		Sleep(1 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	// Handle account linking flow
	err = l.Action().
		UntilTemplateAppears(templates.Cinematic,
			l.Action().
				Sleep(1*time.Second).
				IfTemplateExistsClickPoint(templates.Link, 140, 460).
				IfTemplateExistsClickPoint(templates.Confirm, 203, 364).  // Confirm skip
				IfTemplateExistsClickPoint(templates.Complete, 140, 370), // Complete dialog
			45).
		Execute()

	return err
}

// setupCharacterName handles character name input
func (l *Library) setupCharacterName() error {
	// Click through welcome cutscene until name input
	err := l.Action().
		UntilTemplateAppears(templates.Welcome,
			l.Action().
				Click(253, 506).
				Sleep(110*time.Millisecond),
			45).
		Execute()
	if err != nil {
		return err
	}

	// TODO: waitfortemplate
	// Wait for name input screen
	err = l.Action().
		UntilTemplateAppears(templates.Name,
			l.Action().
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Wait for OK button (keyboard open)
	err = l.Action().
		UntilTemplateAppears(
			templates.OK.InRegion(0, 476, 40, 502),
			l.Action().
				Click(139, 257).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Enter username and submit
	// TODO: Get username from config or generate random
	username := "TestUser123" // Placeholder
	err = l.Action().
		UntilTemplateAppears(templates.Return, l.Action().
			Input(username).
			Sleep(1*time.Second).
			Click(185, 372). // Submit
			Sleep(1*time.Second).
			// Handle rejection (clear and retry)
			Do(func() error {
				if !l.Action().TemplateExists(templates.Return) {
					l.bot.ADB().Click(90, 370)
					l.bot.ADB().Click(139, 254)
					l.bot.ADB().Click(139, 254)
					// TODO: Clear input field
				}
				return nil
			}),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Confirm name
	err = l.Action().
		Sleep(1*time.Second).
		Click(140, 424).
		Sleep(1 * time.Second).
		Execute()

	return err
}

// tutorialFirstPack handles the first pack opening tutorial
func (l *Library) tutorialFirstPack() error {
	// Wait for Pack to be ready
	err := l.Action().
		UntilTemplateAppears(templates.Pack,
			l.Action().
				Click(140, 424).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Swipe to trace the pack
	err = l.Action().
		UntilTemplateDisappears(templates.Pack, l.Action().
			Swipe(adb.SwipeParams{
				X1:       135,
				Y1:       400,
				X2:       135,
				Y2:       200,
				Duration: 100,
			}).
			Sleep(10*time.Millisecond), 45).
		Execute()
	if err != nil {
		return err
	}

	// Click through cards until Swipe indicator appears
	err = l.Action().
		UntilTemplateAppears(templates.Swipe, l.Action().Click(140, 375).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Swipe up through remaining cards
	err = l.Action().
		UntilTemplateDisappears(templates.SwipeUp, l.Action().Swipe(adb.SwipeParams{X1: 266, Y1: 770, X2: 266, Y2: 355, Duration: 60}).
			Sleep(10*time.Millisecond).
			Click(41, 339).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Complete pack opening flow
	err = l.Action().
		Sleep(1*time.Second).
		UntilTemplateAppears(templates.Move, l.Action().Click(134, 375).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	err = l.Action().
		UntilTemplateAppears(templates.Proceed, l.Action().Click(141, 483).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	return l.Action().
		Sleep(1*time.Second).
		Click(204, 371).
		Sleep(1 * time.Second).
		Execute()
}

// tutorialMissionRegistration handles the mission registration part
func (l *Library) tutorialMissionRegistration() error {
	err := l.Action().
		UntilTemplateAppearsClickPoint(templates.Gray, 247, 472, 45).     // Wait for Gray (missions button) to be clickable
		UntilTemplateAppearsClickPoint(templates.Pokeball, 247, 472, 45). // Click through missions until Pokeball appears
		Execute()
	if err != nil {
		return err
	}

	// Click mission accept buttons
	err = l.Action().
		Sleep(1*time.Second).
		Click(141, 294).
		Sleep(1*time.Second).
		Click(141, 294).
		Sleep(1 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	err = l.Action().
		UntilTemplateAppearsClickPoint(templates.Register, 141, 294, 45). // Wait for Register screen
		Sleep(6*time.Second).
		Click(140, 500). // Wait and click through registration
		Sleep(1 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	err = l.Action().
		UntilTemplateAppearsClickPoint(templates.Mission, 143, 360, 45).       // Wait for Mission complete screen
		UntilTemplateAppearsClickPoint(templates.Gray, 143, 360, 45).          // Wait for Gray to be clickable again
		UntilTemplateAppearsClickPoint(templates.Notifications, 145, 194, 45). // Navigate to notifications/packs
		Execute()

	return err
}

// tutorialSecondPack handles the second pack opening tutorial
func (l *Library) tutorialSecondPack() error {
	// Click through to pack
	err := l.Action().
		Sleep(3*time.Second).
		Click(142, 436).
		Sleep(3*time.Second).
		Click(142, 436).
		Sleep(3*time.Second).
		Click(142, 436).
		Sleep(3*time.Second).
		Click(142, 436).
		Sleep(3 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	// Wait for Pack and click Skip
	err = l.Action().
		UntilTemplateAppears(templates.Pack, l.Action().Click(239, 497).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Swipe to trace pack
	err = l.Action().
		UntilTemplateDisappears(templates.Pack, l.Action().Swipe(adb.SwipeParams{
			X1:       135,
			Y1:       400,
			X2:       135,
			Y2:       200,
			Duration: 100,
		}).
			Sleep(10*time.Millisecond).
			Click(41, 339).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Skip through cards until Opening screen
	err = l.Action().
		UntilTemplateAppears(templates.Opening, l.Action().Click(239, 497).
			Sleep(50*time.Millisecond), 45).
		Execute()
	if err != nil {
		return err
	}

	// Click until Skip button appears
	err = l.Action().
		UntilTemplateAppears(templates.Skip, l.Action().Click(146, 496).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Click Next buttons
	err = l.Action().
		UntilTemplateAppears(templates.Next, l.Action().Click(239, 497).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Navigate to Wonder Pick tutorial
	err = l.Action().
		UntilTemplateAppears(templates.Wonder, l.Action().Click(146, 494).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	return l.Action().
		Sleep(3*time.Second).
		Click(140, 358).
		Sleep(1 * time.Second).
		Execute()
}

// tutorialWonderPick handles the Wonder Pick tutorial
func (l *Library) tutorialWonderPick() error {
	// Navigate to main menu
	err := l.Action().
		UntilTemplateAppears(templates.Shop, l.Action().Click(146, 444).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Click Wonder2 icon
	err = l.Action().
		UntilTemplateAppears(templates.Wonder2, l.Action().Click(79, 411).
			Sleep(1*time.Second), 45).
		Execute()
	if err != nil {
		return err
	}

	// Click Wonder3
	err = l.Action().
		UntilTemplateAppears(templates.Wonder3,
			l.Action().
				Click(190, 437).
				Sleep(1*time.Second),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Confirm Wonder Pick selection
	err = l.Action().
		Sleep(2*time.Second).
		UntilTemplateAppears(templates.Wonder4, l.Action().Click(202, 347).
			Sleep(500*time.Millisecond), 45).
		Execute()
	if err != nil {
		return err
	}

	// Start animation and wait
	err = l.Action().
		Sleep(2*time.Second).
		Click(208, 461).
		Sleep(2500 * time.Millisecond). // Wait for animation
		Execute()
	if err != nil {
		return err
	}

	// Wait for Pick screen
	err = l.Action().
		UntilTemplateAppears(templates.Pick, l.Action().Click(208, 461).
			Sleep(350*time.Millisecond),
			45).
		Execute()
	if err != nil {
		return err
	}

	// Select card
	err = l.Action().
		Sleep(1*time.Second).
		Click(187, 345).
		Sleep(1 * time.Second).
		Execute()
	if err != nil {
		return err
	}

	// Click through results until Welcome screen
	err = l.Action().
		UntilTemplateAppears(templates.Welcome, l.Action().Sleep(1*time.Second).
			IfTemplateExistsClick(templates.Skip).
			IfTemplateExistsClickPoint(templates.Next, 146, 494).
			IfTemplateExistsClickPoint(templates.Next2, 146, 494).
			Do(func() error {
				// Fallback clicks if no templates found
				if !l.Action().TemplateExists(templates.Skip) &&
					!l.Action().TemplateExists(templates.Next) &&
					!l.Action().TemplateExists(templates.Next2) &&
					!l.Action().TemplateExists(templates.Welcome) {
					l.bot.ADB().Click(187, 345)
					time.Sleep(1 * time.Second)
					l.bot.ADB().Click(143, 492)
					time.Sleep(1 * time.Second)
					l.bot.ADB().Click(143, 492)
					time.Sleep(1 * time.Second)
				}
				return nil
			}),
			45).
		Execute()

	return err
}
