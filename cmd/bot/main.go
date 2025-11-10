package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"jordanella.com/pocket-tcg-go/internal/config"
	"jordanella.com/pocket-tcg-go/internal/gui"
)

func main() {
	// Create Fyne application
	myApp := app.NewWithID("com.jordanella.pocket-tcg-go")
	myApp.Settings().SetTheme(&gui.BotTheme{})

	// Create main window
	mainWindow := myApp.NewWindow("Pokemon TCG Pocket Bot")
	mainWindow.Resize(gui.DefaultWindowSize)

	// Load configuration
	cfg, err := config.LoadFromINI("Settings.ini", 1)
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		cfg = config.NewDefaultConfig()
	}

	// Create GUI controller
	controller := gui.NewController(cfg, myApp, mainWindow)

	// Build UI with horizontal tabs
	content := controller.BuildUI()

	// Set content and show
	mainWindow.SetContent(content)
	mainWindow.SetMaster()
	mainWindow.ShowAndRun()

	// Cleanup on exit
	controller.Shutdown()
}
