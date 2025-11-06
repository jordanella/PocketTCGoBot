package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/accounts"
)

// AccountTab manages account pool and switching
type AccountTab struct {
	controller *Controller

	// Widgets
	accountsContainer *fyne.Container
	refreshBtn        *widget.Button
	addAccountBtn     *widget.Button
	accountsScroll    *container.Scroll

	// Data
	accountFiles []*accounts.AccountFile
	accountsDir  string
}

// NewAccountTab creates a new account management tab
func NewAccountTab(ctrl *Controller) *AccountTab {
	// Get executable directory
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	exeDir := filepath.Dir(exePath)
	accountsDir := filepath.Join(exeDir, "accounts")

	return &AccountTab{
		controller:   ctrl,
		accountFiles: make([]*accounts.AccountFile, 0),
		accountsDir:  accountsDir,
	}
}

// Build constructs the account management UI
func (a *AccountTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Account Management", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel(fmt.Sprintf("Accounts Directory: %s", a.accountsDir))

	// Buttons
	a.refreshBtn = widget.NewButton("Refresh Accounts", func() {
		a.loadAccounts()
	})

	a.addAccountBtn = widget.NewButton("Add Account", func() {
		a.showAddAccountDialog()
	})

	buttons := container.NewHBox(
		a.refreshBtn,
		a.addAccountBtn,
	)

	// Container for account cards (will be populated dynamically)
	a.accountsContainer = container.NewVBox()

	// Scrollable area for cards
	a.accountsScroll = container.NewVScroll(a.accountsContainer)

	// Initial load
	a.loadAccounts()

	// Layout
	content := container.NewBorder(
		container.NewVBox(
			header,
			subtitle,
			widget.NewSeparator(),
			buttons,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		a.accountsScroll,
	)

	return content
}

// loadAccounts loads all XML accounts from the Accounts directory
func (a *AccountTab) loadAccounts() {
	// Load accounts from XML files
	accountFiles, err := accounts.LoadAccountsFromXML(a.accountsDir)
	if err != nil {
		a.controller.logTab.AddLog(LogLevelError, 0, fmt.Sprintf("Failed to load accounts: %v", err))
		// Only show dialog if window is initialized
		if a.controller.window != nil {
			dialog.ShowError(fmt.Errorf("failed to load accounts: %v", err), a.controller.window)
		}
		return
	}

	a.accountFiles = accountFiles

	// Clear existing cards
	a.accountsContainer.Objects = []fyne.CanvasObject{}

	// Create cards for each account
	if len(accountFiles) == 0 {
		emptyLabel := widget.NewLabel("No accounts found. Click 'Add Account' to create one.")
		emptyLabel.Alignment = fyne.TextAlignCenter
		a.accountsContainer.Add(emptyLabel)
	} else {
		for _, accountFile := range accountFiles {
			card := a.createAccountCard(accountFile)
			a.accountsContainer.Add(card)
		}
	}

	a.accountsContainer.Refresh()
	a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Loaded %d accounts", len(accountFiles)))
}

// createAccountCard creates a card widget for an account
func (a *AccountTab) createAccountCard(accountFile *accounts.AccountFile) fyne.CanvasObject {
	// Card background
	cardBg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
	cardBg.SetMinSize(fyne.NewSize(0, 120))

	// Filename label (bold and larger)
	filenameLabel := widget.NewLabelWithStyle(
		accountFile.Filename,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	// Device Account info
	deviceAccountLabel := widget.NewLabel("Device Account:")
	deviceAccountValue := widget.NewLabel(accountFile.DeviceAccount)

	// Device Password info
	devicePasswordLabel := widget.NewLabel("Device Password:")
	devicePasswordValue := widget.NewLabel(accountFile.DevicePassword)

	// Delete button
	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		a.showDeleteConfirmation(accountFile)
	})

	// Edit button
	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		a.showEditAccountDialog(accountFile)
	})

	// Inject button
	injectBtn := widget.NewButtonWithIcon("Inject", theme.UploadIcon(), func() {
		a.showInjectAccountDialog(accountFile)
	})

	// Layout for card content
	infoGrid := container.New(
		layout.NewFormLayout(),
		deviceAccountLabel,
		deviceAccountValue,
		devicePasswordLabel,
		devicePasswordValue,
	)

	buttonBar := container.NewHBox(
		layout.NewSpacer(),
		injectBtn,
		editBtn,
		deleteBtn,
	)

	cardContent := container.NewBorder(
		filenameLabel,
		buttonBar,
		nil,
		nil,
		container.NewPadded(infoGrid),
	)

	// Stack background and content
	card := container.NewStack(
		cardBg,
		container.NewPadded(cardContent),
	)

	return card
}

// showAddAccountDialog shows dialog to add a new account
func (a *AccountTab) showAddAccountDialog() {
	// Create input fields
	filenameEntry := widget.NewEntry()
	filenameEntry.SetPlaceHolder("e.g., account1.xml")

	deviceAccountEntry := widget.NewEntry()
	deviceAccountEntry.SetPlaceHolder("Enter device account...")

	devicePasswordEntry := widget.NewEntry()
	devicePasswordEntry.SetPlaceHolder("Enter device password...")

	// Create form
	items := []*widget.FormItem{
		{Text: "Filename", Widget: filenameEntry},
		{Text: "Device Account", Widget: deviceAccountEntry},
		{Text: "Device Password", Widget: devicePasswordEntry},
	}

	// Show dialog
	dialog.ShowForm("Add Account", "Add", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}

		filename := filenameEntry.Text
		deviceAccount := deviceAccountEntry.Text
		devicePassword := devicePasswordEntry.Text

		// Validate inputs
		if filename == "" || deviceAccount == "" || devicePassword == "" {
			dialog.ShowError(fmt.Errorf("all fields are required"), a.controller.window)
			return
		}

		// Ensure filename ends with .xml
		if filepath.Ext(filename) != ".xml" {
			filename += ".xml"
		}

		// Save account to XML
		if err := accounts.SaveAccountToXML(a.accountsDir, filename, deviceAccount, devicePassword); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save account: %v", err), a.controller.window)
			a.controller.logTab.AddLog(LogLevelError, 0, fmt.Sprintf("Failed to save account: %v", err))
			return
		}

		// Reload accounts
		a.loadAccounts()
		dialog.ShowInformation("Success", fmt.Sprintf("Account %s added successfully", filename), a.controller.window)
		a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Account %s added", filename))
	}, a.controller.window)
}

// showEditAccountDialog shows dialog to edit an existing account
func (a *AccountTab) showEditAccountDialog(accountFile *accounts.AccountFile) {
	// Create input fields with current values
	deviceAccountEntry := widget.NewEntry()
	deviceAccountEntry.SetText(accountFile.DeviceAccount)

	devicePasswordEntry := widget.NewEntry()
	devicePasswordEntry.SetText(accountFile.DevicePassword)

	// Create form
	items := []*widget.FormItem{
		{Text: "Filename", Widget: widget.NewLabel(accountFile.Filename)},
		{Text: "Device Account", Widget: deviceAccountEntry},
		{Text: "Device Password", Widget: devicePasswordEntry},
	}

	// Show dialog
	dialog.ShowForm("Edit Account", "Save", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}

		deviceAccount := deviceAccountEntry.Text
		devicePassword := devicePasswordEntry.Text

		// Validate inputs
		if deviceAccount == "" || devicePassword == "" {
			dialog.ShowError(fmt.Errorf("all fields are required"), a.controller.window)
			return
		}

		// Save account to XML (overwrites existing file)
		if err := accounts.SaveAccountToXML(a.accountsDir, accountFile.Filename, deviceAccount, devicePassword); err != nil {
			dialog.ShowError(fmt.Errorf("failed to update account: %v", err), a.controller.window)
			a.controller.logTab.AddLog(LogLevelError, 0, fmt.Sprintf("Failed to update account: %v", err))
			return
		}

		// Reload accounts
		a.loadAccounts()
		dialog.ShowInformation("Success", fmt.Sprintf("Account %s updated successfully", accountFile.Filename), a.controller.window)
		a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Account %s updated", accountFile.Filename))
	}, a.controller.window)
}

// showDeleteConfirmation shows confirmation dialog before deleting an account
func (a *AccountTab) showDeleteConfirmation(accountFile *accounts.AccountFile) {
	dialog.ShowConfirm(
		"Confirm Deletion",
		fmt.Sprintf("Are you sure you want to delete account:\n%s\n\nDevice Account: %s\n\nThis action cannot be undone.",
			accountFile.Filename, accountFile.DeviceAccount),
		func(ok bool) {
			if !ok {
				return
			}

			// Delete XML file
			if err := accounts.DeleteAccountXML(accountFile.FilePath); err != nil {
				dialog.ShowError(fmt.Errorf("failed to delete account: %v", err), a.controller.window)
				a.controller.logTab.AddLog(LogLevelError, 0, fmt.Sprintf("Failed to delete account: %v", err))
				return
			}

			// Reload accounts
			a.loadAccounts()
			dialog.ShowInformation("Success", fmt.Sprintf("Account %s deleted successfully", accountFile.Filename), a.controller.window)
			a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Account %s deleted", accountFile.Filename))
		},
		a.controller.window,
	)
}

// showInjectAccountDialog shows dialog to inject account into an instance
func (a *AccountTab) showInjectAccountDialog(accountFile *accounts.AccountFile) {
	cfg := a.controller.GetConfig()

	// Get available instances
	adbPath := cfg.ADB().Path
	if adbPath == "" {
		dialog.ShowError(fmt.Errorf("%s", "ADB path not configured. Please configure it in the Configuration tab."), a.controller.window)
		return
	}

	// Get instance configurations to build dropdown
	mgr := a.controller.CreateEmulatorManager()

	// Log discovery attempt
	a.controller.logTab.AddLog(LogLevelInfo, 0, "Discovering running instances...")

	// Discover running instances
	if err := mgr.DiscoverInstances(); err != nil {
		dialog.ShowError(fmt.Errorf("failed to discover instances: %v", err), a.controller.window)
		a.controller.logTab.AddLog(LogLevelError, 0, fmt.Sprintf("Failed to discover instances: %v", err))
		return
	}

	instances := mgr.GetAllInstances()
	a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Found %d running instances", len(instances)))

	if len(instances) == 0 {
		dialog.ShowError(fmt.Errorf("%s", "no running instances found. Please start at least one MuMu instance."), a.controller.window)
		return
	}

	// Build instance options for dropdown
	instanceOptions := make([]string, 0, len(instances))
	instanceMap := make(map[string]int)

	for _, inst := range instances {
		// Display window title and port
		windowTitle := "Unknown"
		adbPort := 0

		if inst.MuMu != nil {
			windowTitle = inst.MuMu.WindowTitle
			adbPort = inst.MuMu.ADBPort
		}

		displayName := fmt.Sprintf("Window '%s' (Port %d)", windowTitle, adbPort)
		instanceOptions = append(instanceOptions, displayName)
		instanceMap[displayName] = inst.Index

		// Log what we detected
		a.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Detected: Window '%s' Port %d -> Index %d", windowTitle, adbPort, inst.Index))
	}

	// Create instance selector
	instanceSelect := widget.NewSelect(instanceOptions, nil)
	if len(instanceOptions) > 0 {
		instanceSelect.SetSelected(instanceOptions[0])
	}

	// Create form
	items := []*widget.FormItem{
		{Text: "Account File", Widget: widget.NewLabel(accountFile.Filename)},
		{Text: "Device Account", Widget: widget.NewLabel(accountFile.DeviceAccount)},
		{Text: "Target Instance", Widget: instanceSelect},
	}

	// Show dialog
	dialog.ShowForm("Inject Account", "Inject", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}

		selectedInstance := instanceSelect.Selected
		if selectedInstance == "" {
			dialog.ShowError(fmt.Errorf("please select an instance"), a.controller.window)
			return
		}

		instanceIndex, exists := instanceMap[selectedInstance]
		if !exists {
			dialog.ShowError(fmt.Errorf("invalid instance selection"), a.controller.window)
			return
		}

		// Get the instance
		inst, err := mgr.GetInstance(instanceIndex)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get instance: %v", err), a.controller.window)
			return
		}

		// Perform injection in a goroutine
		go func() {
			a.controller.logTab.AddLog(LogLevelInfo, instanceIndex, fmt.Sprintf("Injecting account: %s", accountFile.Filename))

			// Inject the account
			err := accounts.InjectAccount(adbPath, inst.MuMu.ADBPort, accountFile.FilePath)

			// Update UI on main thread
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(fmt.Errorf("failed to inject account: %v", err), a.controller.window)
					a.controller.logTab.AddLog(LogLevelError, instanceIndex, fmt.Sprintf("Injection failed: %v", err))
					return
				}

				dialog.ShowInformation("Success", fmt.Sprintf("Account %s successfully injected into instance %d", accountFile.Filename, instanceIndex), a.controller.window)
				a.controller.logTab.AddLog(LogLevelInfo, instanceIndex, fmt.Sprintf("Account %s injected successfully", accountFile.Filename))
			})
		}()
	}, a.controller.window)
}

// GetAccountFiles returns all loaded account files
func (a *AccountTab) GetAccountFiles() []*accounts.AccountFile {
	return a.accountFiles
}
