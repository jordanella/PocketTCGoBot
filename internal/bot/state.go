package bot

import "time"

// All state-related types
type State struct {
	// Current account being used
	LoadedAccount      *AccountState
	LoadedAccountIndex int

	// Session statistics
	Rerolls         int
	RerollStartTime time.Time
	PacksInPool     int
	PacksThisRun    int

	// Current pack state
	FoundGodPack       bool
	CurrentPackType    string // "Mewtwo", "Charizard", etc.
	CurrentPackIs6Card bool
	CurrentPackIs4Card bool

	// Resource limits
	CantOpenMorePacks bool
	MaxAccountPackNum int

	// Friends
	Friended bool
	WPThanksSavedUsername  string
	WPThanksSavedFriendCode string
}

// AccountState tracks per-account metadata
type AccountState struct {
	FileName         string
	FileNameOrig     string // Original filename before temp rename
	FileNameTmp      string // Temporary filename during processing
	OpenPacks        int
	HasPackInfo      bool
	ModifiedTime     time.Time

	// Mission completion tracking
	Missions MissionTracker

	// Metadata flags (embedded in XML filename)
	Flags    []string // e.g., "W" for wonderpick-eligible, "T" for testing
	Username string
	FriendCode string

	// Tracking
	LastUsedTime time.Time
	TimesUsed    int
}

type MissionTracker struct {
	BeginnerDone      bool
	SoloBattleDone    bool
	IntermediateDone  bool
	SpecialDone       bool
	ResetSpecialDone  bool
	HasPackInTesting  bool
}

// State management methods

// saveState saves the bot state to disk
func (b *Bot) saveState() error {
	// TODO: Implement state persistence
	// Could save to JSON file or database
	// Include account state, statistics, progress
	return nil
}

// loadState loads the bot state from disk
func (b *Bot) loadState() error {
	// TODO: Implement state loading
	// Restore previous session state if exists
	// Initialize new state if doesn't exist
	return nil
}

// resetState resets the bot state to initial values
func (b *Bot) resetState() {
	// Reset statistics
	b.state.Rerolls = 0
	b.state.PacksInPool = 0
	b.state.PacksThisRun = 0

	// Reset pack state
	b.state.FoundGodPack = false
	b.state.CurrentPackType = ""
	b.state.CurrentPackIs6Card = false
	b.state.CurrentPackIs4Card = false

	// Reset flags
	b.state.CantOpenMorePacks = false
	b.state.Friended = false

	// Reset account state
	b.state.ResetAccountState()
}

// Account state helpers
func (s *State) ResetAccountState() {
	s.LoadedAccount = nil
	s.LoadedAccountIndex = 0
}

func (s *State) IncrementPackCount() {
	if s.LoadedAccount != nil {
		s.LoadedAccount.OpenPacks++
	}
	s.PacksInPool++
	s.PacksThisRun++
}
