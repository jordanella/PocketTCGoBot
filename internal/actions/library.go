package actions

// ActionLibrary provides high-level reusable actions
type Library struct {
	bot BotInterface
}

func NewLibrary(b BotInterface) *Library { return &Library{bot: b} }

// Action returns a new ActionBuilder initialized with the bot reference
// This allows for easy chaining like: l.Action().Click(x, y).Sleep(1*time.Second).Execute()
func (l *Library) Action() *ActionBuilder {
	return &ActionBuilder{bot: l.bot}
}

// These are composed actions that can be called directly or chained
func (l *Library) GoHome() error {
	// determine whether level up checks should be in this or it falls under a monitoring situation
	return nil
}

func (l *Library) GoToGifts() error    { return nil }
func (l *Library) GoToBattle() error   { return nil }
func (l *Library) GoToDex() error      { return nil }
func (l *Library) GoToMissions() error { return nil }

func (l *Library) EraseInput() error { return nil } // purpose?

func (l *Library) LevelUp() error {
	/*
	   builder := &actions.ActionBuilder{}

	   return builder.
	       IfTemplateExists(templates.LevelUp, func(ab *actions.ActionBuilder) {
	           ab.FindAndClickCenter(templates.Button.InRegion(75, 340, 195, 530, 80), 0).
	           Sleep(500 * time.Millisecond)
	       }).
	       Execute()
	*/
	return nil
}

// etc.
/*
actionLib.NewAction().
    IfTemplateExists("templates/popup.png", 0.85, func(ab *actions.ActionBuilder) {
        // Close popup if it exists
        ab.FindAndClickCenter("templates/close_button.png", 0.85).
           Sleep(500 * time.Millisecond)
    }).
    Execute()
```go
func (l *Library) CompleteWonderPick() error {
    builder := &actions.ActionBuilder{}

    return builder.
        // Wait for wonder pick screen
        WaitFor(templates.WonderPickTemplates.Screen, 10*time.Second).

        // Click the first card
        Click(200, 500).
        Sleep(500*time.Millisecond).

        // Wait for card reveal
        WaitFor(templates.WonderPickTemplates.CardReveal, 5*time.Second).

        // Click to claim
        FindAndClick(templates.WonderPickTemplates.ClaimButton, 0, 0).

        // Wait and click through results
        Sleep(2*time.Second).
        Click(540, 1600).

        Execute()
}
```LevelUp() {
    Leveled := FindOrLoseImage(100, 86, 167, 116, , "LevelUp", 0)
    if(Leveled) {
        clickButton := FindOrLoseImage(75, 340, 195, 530, 80, "Button", 0, failSafeTime)
        StringSplit, pos, clickButton, `,  ; Split at ", "
        if (scaleParam = 287) {
            pos2 += 5
        }
        adbClick_wbb(pos1, pos2)
    }
    Delay(1)
}
*/
