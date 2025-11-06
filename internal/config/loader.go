package config

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// LoadFromINI loads configuration from Settings.ini file
func LoadFromINI(path string, instance int) (*bot.Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	section := cfg.Section("UserSettings")

	config := &bot.Config{
		Instance:        instance,
		EnabledPacks:    make(map[string]bool),
		ShinyPacks:      make(map[string]bool),
		MinStarsPerPack: make(map[string]int),
	}

	// Instance configuration
	config.Columns = section.Key("Columns").MustInt(5)
	config.RowGap = section.Key("rowGap").MustInt(100)
	config.SelectedMonitor = section.Key("SelectedMonitorIndex").MustInt(1)
	config.DefaultLanguage = section.Key("defaultLanguage").MustString("Scale125")
	config.FolderPath = section.Key("folderPath").MustString("C:\\Program Files\\Netease")

	// Delete method
	deleteMethodStr := section.Key("deleteMethod").MustString("Create Bots (13P)")
	config.DeleteMethod = parseDeleteMethod(deleteMethodStr)

	// Injection settings
	config.InjectSortMethod = parseSortMethod(section.Key("injectSortMethod").MustString("ModifiedAsc"))
	config.InjectMinPacks = section.Key("injectMinPacks").MustInt(0)
	config.InjectMaxPacks = section.Key("injectMaxPacks").MustInt(39)

	// Account waiting
	config.WaitForEligibleAccounts = section.Key("waitForEligibleAccounts").MustBool(true)
	config.MaxWaitHours = section.Key("maxWaitHours").MustInt(24)

	// Pack preferences
	packList := []string{
		"Mewtwo", "Charizard", "Pikachu", "Mew",
		"Dialga", "Palkia", "Arceus", "Shining",
		"Solgaleo", "Lunala", "Buzzwole", "Eevee",
		"HoOh", "Lugia", "Springs", "Deluxe",
		"MegaGyarados", "MegaBlaziken", "MegaAltaria",
	}

	for _, pack := range packList {
		config.EnabledPacks[pack] = section.Key(pack).MustBool(false)
	}

	// Shiny packs
	shinyList := []string{
		"Shining", "Solgaleo", "Lunala", "Buzzwole", "Eevee",
		"HoOh", "Lugia", "Springs", "Deluxe",
		"MegaGyarados", "MegaBlaziken", "MegaAltaria",
	}
	for _, pack := range shinyList {
		config.ShinyPacks[pack] = true
	}

	// Star requirements
	config.MinStars = section.Key("minStars").MustInt(0)
	config.MinStarsShiny = section.Key("minStarsShiny").MustInt(0)

	// Per-pack star requirements
	config.MinStarsPerPack["A1Mewtwo"] = section.Key("minStarsA1Mewtwo").MustInt(0)
	config.MinStarsPerPack["A1Charizard"] = section.Key("minStarsA1Charizard").MustInt(0)
	config.MinStarsPerPack["A1Pikachu"] = section.Key("minStarsA1Pikachu").MustInt(0)
	config.MinStarsPerPack["A1a"] = section.Key("minStarsA1a").MustInt(0)
	config.MinStarsPerPack["A2Dialga"] = section.Key("minStarsA2Dialga").MustInt(0)
	config.MinStarsPerPack["A2Palkia"] = section.Key("minStarsA2Palkia").MustInt(0)
	config.MinStarsPerPack["A2a"] = section.Key("minStarsA2a").MustInt(0)
	config.MinStarsPerPack["A2b"] = section.Key("minStarsA2b").MustInt(0)
	config.MinStarsPerPack["A3Solgaleo"] = section.Key("minStarsA3Solgaleo").MustInt(0)
	config.MinStarsPerPack["A3Lunala"] = section.Key("minStarsA3Lunala").MustInt(0)
	config.MinStarsPerPack["A3a"] = section.Key("minStarsA3a").MustInt(0)
	config.MinStarsPerPack["A4HoOh"] = section.Key("minStarsA4HoOh").MustInt(0)
	config.MinStarsPerPack["A4Lugia"] = section.Key("minStarsA4Lugia").MustInt(0)
	config.MinStarsPerPack["A4Springs"] = section.Key("minStarsA4Springs").MustInt(0)
	config.MinStarsPerPack["A4Deluxe"] = section.Key("minStarsA4Deluxe").MustInt(0)
	config.MinStarsPerPack["MegaGyarados"] = section.Key("minStarsMegaGyarados").MustInt(0)
	config.MinStarsPerPack["MegaBlaziken"] = section.Key("minStarsMegaBlaziken").MustInt(0)
	config.MinStarsPerPack["MegaAltaria"] = section.Key("minStarsMegaAltaria").MustInt(0)

	// Pack validation criteria
	config.CheckShinyPackOnly = section.Key("CheckShinyPackOnly").MustBool(false)
	config.TrainerCheck = section.Key("TrainerCheck").MustBool(false)
	config.FullArtCheck = section.Key("FullArtCheck").MustBool(false)
	config.RainbowCheck = section.Key("RainbowCheck").MustBool(false)
	config.ShinyCheck = section.Key("ShinyCheck").MustBool(false)
	config.CrownCheck = section.Key("CrownCheck").MustBool(false)
	config.ImmersiveCheck = section.Key("ImmersiveCheck").MustBool(false)
	config.InvalidCheck = section.Key("InvalidCheck").MustBool(false)
	config.PseudoGodPack = section.Key("PseudoGodPack").MustBool(false)

	// Missions
	config.SkipMissionsInjectMissions = section.Key("skipMissionsInjectMissions").MustBool(false)
	config.ClaimSpecialMissions = section.Key("claimSpecialMissions").MustBool(false)
	config.ClaimDailyMission = section.Key("claimDailyMission").MustBool(false)
	config.WonderpickForEventMissions = section.Key("wonderpickForEventMissions").MustBool(false)

	// Resources
	config.SpendHourGlass = section.Key("spendHourGlass").MustBool(true)
	config.OpenExtraPack = section.Key("openExtraPack").MustBool(false)

	// Social
	config.FriendID = section.Key("FriendID").MustString("")
	config.CheckWPThanks = section.Key("checkWPthanks").MustBool(false)
	config.ShowcaseEnabled = section.Key("showcaseEnabled").MustBool(false)

	// Parse friend IDs (comma-separated list)
	friendIDsStr := section.Key("FriendIDs").MustString("")
	if friendIDsStr != "" {
		config.FriendIDs = strings.Split(friendIDsStr, ",")
		for i := range config.FriendIDs {
			config.FriendIDs[i] = strings.TrimSpace(config.FriendIDs[i])
		}
	}

	// S4T Integration
	config.S4TEnabled = section.Key("s4tEnabled").MustBool(false)
	config.S4TSilent = section.Key("s4tSilent").MustBool(true)
	config.S4T3Diamond = section.Key("s4t3Dmnd").MustBool(false)
	config.S4T4Diamond = section.Key("s4t4Dmnd").MustBool(false)
	config.S4T1Star = section.Key("s4t1Star").MustBool(false)
	config.S4TGholdengo = section.Key("s4tGholdengo").MustBool(false)
	config.S4TTrainer = section.Key("s4tTrainer").MustBool(false)
	config.S4TRainbow = section.Key("s4tRainbow").MustBool(false)
	config.S4TFullArt = section.Key("s4tFullArt").MustBool(false)
	config.S4TCrown = section.Key("s4tCrown").MustBool(false)
	config.S4TImmersive = section.Key("s4tImmersive").MustBool(false)
	config.S4TShiny1Star = section.Key("s4tShiny1Star").MustBool(false)
	config.S4TShiny2Star = section.Key("s4tShiny2Star").MustBool(false)
	config.S4TWonderPick = section.Key("s4tWP").MustBool(false)
	config.S4TWPMinCards = section.Key("s4tWPMinCards").MustInt(1)
	config.S4TDiscordWebhook = section.Key("s4tDiscordWebhookURL").MustString("")
	config.S4TDiscordUserID = section.Key("s4tDiscordUserId").MustString("")
	config.S4TSendAccountXml = section.Key("s4tSendAccountXml").MustBool(true)

	// OCR
	config.OCRLanguage = section.Key("ocrLanguage").MustString("en")
	config.OCRShinedust = section.Key("ocrShinedust").MustBool(false)

	// Behavior
	godPackStr := section.Key("godPack").MustString("Continue")
	config.GodPackAction = parseGodPackAction(godPackStr)
	config.PackMethod = section.Key("packMethod").MustInt(0)
	config.NukeAccount = section.Key("nukeAccount").MustBool(false)
	config.RunMain = section.Key("runMain").MustBool(true)
	config.Mains = section.Key("Mains").MustInt(1)

	// Performance
	config.Delay = section.Key("Delay").MustInt(250)
	config.SwipeSpeed = section.Key("swipeSpeed").MustInt(300)
	config.SlowMotion = section.Key("slowMotion").MustBool(false)
	config.WaitTime = section.Key("waitTime").MustInt(5)

	// Display
	config.ShowStatus = section.Key("showStatus").MustBool(true)

	// Debug
	config.VerboseLogging = section.Key("debugMode").MustBool(false)

	// Extended configuration (new fields for GUI and advanced features)
	config.ADBPath = section.Key("adbPath").MustString("")
	config.MuMuWindowWidth = section.Key("mumuWindowWidth").MustInt(0)
	config.MuMuWindowHeight = section.Key("mumuWindowHeight").MustInt(0)
	config.LogLevel = section.Key("logLevel").MustString("INFO")
	config.LoggingEnabled = section.Key("loggingEnabled").MustBool(true)

	// Load instance-specific settings
	instanceSection := cfg.Section(fmt.Sprintf("Instance%d", instance))
	if instanceSection != nil {
		config.DeadCheck = instanceSection.Key("DeadCheck").MustBool(false)
	}

	return config, nil
}

func parseDeleteMethod(s string) bot.DeleteMethod {
	switch s {
	case "Create Bots (13P)":
		return bot.DeleteMethodCreateBots
	case "Inject 13P+":
		return bot.DeleteMethodInject13P
	case "Inject Wonderpick 96P+":
		return bot.DeleteMethodInjectWonderPick96P
	case "Inject Missions":
		return bot.DeleteMethodInjectMissions
	default:
		return bot.DeleteMethodCreateBots
	}
}

func parseSortMethod(s string) bot.SortMethod {
	switch s {
	case "ModifiedAsc":
		return bot.SortMethodModifiedAsc
	case "ModifiedDesc":
		return bot.SortMethodModifiedDesc
	case "PacksAsc":
		return bot.SortMethodPacksAsc
	case "PacksDesc":
		return bot.SortMethodPacksDesc
	default:
		return bot.SortMethodModifiedAsc
	}
}

func parseGodPackAction(s string) bot.GodPackAction {
	switch s {
	case "Close":
		return bot.GodPackClose
	case "Pause":
		return bot.GodPackPause
	case "Continue":
		return bot.GodPackContinue
	default:
		return bot.GodPackContinue
	}
}

// GetEnabledPacks returns list of enabled pack names
func GetEnabledPacks(config *bot.Config) []string {
	enabled := []string{}
	for pack, isEnabled := range config.EnabledPacks {
		if isEnabled {
			enabled = append(enabled, pack)
		}
	}
	return enabled
}

// IsShinyPack checks if a pack is a shiny pack type
func IsShinyPack(config *bot.Config, packName string) bool {
	return config.ShinyPacks[packName]
}

// GetMinStarsForPack gets minimum star requirement for a specific pack
func GetMinStarsForPack(config *bot.Config, packName string) int {
	// Check pack-specific requirement first
	if minStars, ok := config.MinStarsPerPack[packName]; ok && minStars > 0 {
		return minStars
	}

	// Fall back to global requirement
	if IsShinyPack(config, packName) && config.MinStarsShiny > 0 {
		return config.MinStarsShiny
	}

	return config.MinStars
}

// NewDefaultConfig creates a config with default values
func NewDefaultConfig() *bot.Config {
	return &bot.Config{
		Instance:         1,
		EnabledPacks:     make(map[string]bool),
		ShinyPacks:       make(map[string]bool),
		MinStarsPerPack:  make(map[string]int),
		Columns:          5,
		RowGap:           100,
		Delay:            250,
		SwipeSpeed:       300,
		WaitTime:         5,
		FolderPath:       "C:\\Program Files\\Netease\\MuMuPlayer-12.0",
		DefaultLanguage:  "Scale125",
		ADBPath:          "",
		MuMuWindowWidth:  540,
		MuMuWindowHeight: 960,
		LogLevel:         "INFO",
		LoggingEnabled:   true,
		VerboseLogging:   false,
	}
}

// SaveToINI saves configuration to an INI file
func SaveToINI(config *bot.Config, path string) error {
	cfg := ini.Empty()
	section := cfg.Section("UserSettings")

	// Instance configuration
	section.Key("Columns").SetValue(fmt.Sprintf("%d", config.Columns))
	section.Key("rowGap").SetValue(fmt.Sprintf("%d", config.RowGap))
	section.Key("SelectedMonitorIndex").SetValue(fmt.Sprintf("%d", config.SelectedMonitor))
	section.Key("defaultLanguage").SetValue(config.DefaultLanguage)
	section.Key("folderPath").SetValue(config.FolderPath)

	// Delete method
	section.Key("deleteMethod").SetValue(config.DeleteMethod.String())

	// Injection settings
	section.Key("injectSortMethod").SetValue(config.InjectSortMethod.String())
	section.Key("injectMinPacks").SetValue(fmt.Sprintf("%d", config.InjectMinPacks))
	section.Key("injectMaxPacks").SetValue(fmt.Sprintf("%d", config.InjectMaxPacks))

	// Account waiting
	section.Key("waitForEligibleAccounts").SetValue(fmt.Sprintf("%t", config.WaitForEligibleAccounts))
	section.Key("maxWaitHours").SetValue(fmt.Sprintf("%d", config.MaxWaitHours))

	// Pack preferences
	packList := []string{
		"Mewtwo", "Charizard", "Pikachu", "Mew",
		"Dialga", "Palkia", "Arceus", "Shining",
		"Solgaleo", "Lunala", "Buzzwole", "Eevee",
		"HoOh", "Lugia", "Springs", "Deluxe",
		"MegaGyarados", "MegaBlaziken", "MegaAltaria",
	}
	for _, pack := range packList {
		enabled := config.EnabledPacks[pack]
		section.Key(pack).SetValue(fmt.Sprintf("%t", enabled))
	}

	// Star requirements
	section.Key("minStars").SetValue(fmt.Sprintf("%d", config.MinStars))
	section.Key("minStarsShiny").SetValue(fmt.Sprintf("%d", config.MinStarsShiny))

	// Per-pack star requirements
	section.Key("minStarsA1Mewtwo").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A1Mewtwo"]))
	section.Key("minStarsA1Charizard").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A1Charizard"]))
	section.Key("minStarsA1Pikachu").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A1Pikachu"]))
	section.Key("minStarsA1a").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A1a"]))
	section.Key("minStarsA2Dialga").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A2Dialga"]))
	section.Key("minStarsA2Palkia").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A2Palkia"]))
	section.Key("minStarsA2a").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A2a"]))
	section.Key("minStarsA2b").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A2b"]))
	section.Key("minStarsA3Solgaleo").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A3Solgaleo"]))
	section.Key("minStarsA3Lunala").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A3Lunala"]))
	section.Key("minStarsA3a").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A3a"]))
	section.Key("minStarsA4HoOh").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A4HoOh"]))
	section.Key("minStarsA4Lugia").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A4Lugia"]))
	section.Key("minStarsA4Springs").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A4Springs"]))
	section.Key("minStarsA4Deluxe").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["A4Deluxe"]))
	section.Key("minStarsMegaGyarados").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["MegaGyarados"]))
	section.Key("minStarsMegaBlaziken").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["MegaBlaziken"]))
	section.Key("minStarsMegaAltaria").SetValue(fmt.Sprintf("%d", config.MinStarsPerPack["MegaAltaria"]))

	// Pack validation criteria
	section.Key("CheckShinyPackOnly").SetValue(fmt.Sprintf("%t", config.CheckShinyPackOnly))
	section.Key("TrainerCheck").SetValue(fmt.Sprintf("%t", config.TrainerCheck))
	section.Key("FullArtCheck").SetValue(fmt.Sprintf("%t", config.FullArtCheck))
	section.Key("RainbowCheck").SetValue(fmt.Sprintf("%t", config.RainbowCheck))
	section.Key("ShinyCheck").SetValue(fmt.Sprintf("%t", config.ShinyCheck))
	section.Key("CrownCheck").SetValue(fmt.Sprintf("%t", config.CrownCheck))
	section.Key("ImmersiveCheck").SetValue(fmt.Sprintf("%t", config.ImmersiveCheck))
	section.Key("InvalidCheck").SetValue(fmt.Sprintf("%t", config.InvalidCheck))
	section.Key("PseudoGodPack").SetValue(fmt.Sprintf("%t", config.PseudoGodPack))

	// Missions
	section.Key("skipMissionsInjectMissions").SetValue(fmt.Sprintf("%t", config.SkipMissionsInjectMissions))
	section.Key("claimSpecialMissions").SetValue(fmt.Sprintf("%t", config.ClaimSpecialMissions))
	section.Key("claimDailyMission").SetValue(fmt.Sprintf("%t", config.ClaimDailyMission))
	section.Key("wonderpickForEventMissions").SetValue(fmt.Sprintf("%t", config.WonderpickForEventMissions))

	// Resources
	section.Key("spendHourGlass").SetValue(fmt.Sprintf("%t", config.SpendHourGlass))
	section.Key("openExtraPack").SetValue(fmt.Sprintf("%t", config.OpenExtraPack))

	// Social
	section.Key("FriendID").SetValue(config.FriendID)
	section.Key("checkWPthanks").SetValue(fmt.Sprintf("%t", config.CheckWPThanks))
	section.Key("showcaseEnabled").SetValue(fmt.Sprintf("%t", config.ShowcaseEnabled))

	// Friend IDs (comma-separated list)
	if len(config.FriendIDs) > 0 {
		section.Key("FriendIDs").SetValue(strings.Join(config.FriendIDs, ","))
	}

	// S4T Integration
	section.Key("s4tEnabled").SetValue(fmt.Sprintf("%t", config.S4TEnabled))
	section.Key("s4tSilent").SetValue(fmt.Sprintf("%t", config.S4TSilent))
	section.Key("s4t3Dmnd").SetValue(fmt.Sprintf("%t", config.S4T3Diamond))
	section.Key("s4t4Dmnd").SetValue(fmt.Sprintf("%t", config.S4T4Diamond))
	section.Key("s4t1Star").SetValue(fmt.Sprintf("%t", config.S4T1Star))
	section.Key("s4tGholdengo").SetValue(fmt.Sprintf("%t", config.S4TGholdengo))
	section.Key("s4tTrainer").SetValue(fmt.Sprintf("%t", config.S4TTrainer))
	section.Key("s4tRainbow").SetValue(fmt.Sprintf("%t", config.S4TRainbow))
	section.Key("s4tFullArt").SetValue(fmt.Sprintf("%t", config.S4TFullArt))
	section.Key("s4tCrown").SetValue(fmt.Sprintf("%t", config.S4TCrown))
	section.Key("s4tImmersive").SetValue(fmt.Sprintf("%t", config.S4TImmersive))
	section.Key("s4tShiny1Star").SetValue(fmt.Sprintf("%t", config.S4TShiny1Star))
	section.Key("s4tShiny2Star").SetValue(fmt.Sprintf("%t", config.S4TShiny2Star))
	section.Key("s4tWP").SetValue(fmt.Sprintf("%t", config.S4TWonderPick))
	section.Key("s4tWPMinCards").SetValue(fmt.Sprintf("%d", config.S4TWPMinCards))
	section.Key("s4tDiscordWebhookURL").SetValue(config.S4TDiscordWebhook)
	section.Key("s4tDiscordUserId").SetValue(config.S4TDiscordUserID)
	section.Key("s4tSendAccountXml").SetValue(fmt.Sprintf("%t", config.S4TSendAccountXml))

	// OCR
	section.Key("ocrLanguage").SetValue(config.OCRLanguage)
	section.Key("ocrShinedust").SetValue(fmt.Sprintf("%t", config.OCRShinedust))

	// Behavior
	section.Key("godPack").SetValue(config.GodPackAction.String())
	section.Key("packMethod").SetValue(fmt.Sprintf("%d", config.PackMethod))
	section.Key("nukeAccount").SetValue(fmt.Sprintf("%t", config.NukeAccount))
	section.Key("runMain").SetValue(fmt.Sprintf("%t", config.RunMain))
	section.Key("Mains").SetValue(fmt.Sprintf("%d", config.Mains))

	// Performance
	section.Key("Delay").SetValue(fmt.Sprintf("%d", config.Delay))
	section.Key("swipeSpeed").SetValue(fmt.Sprintf("%d", config.SwipeSpeed))
	section.Key("slowMotion").SetValue(fmt.Sprintf("%t", config.SlowMotion))
	section.Key("waitTime").SetValue(fmt.Sprintf("%d", config.WaitTime))

	// Display
	section.Key("showStatus").SetValue(fmt.Sprintf("%t", config.ShowStatus))

	// Debug
	section.Key("debugMode").SetValue(fmt.Sprintf("%t", config.VerboseLogging))

	// Extended configuration (new fields for GUI and advanced features)
	section.Key("adbPath").SetValue(config.ADBPath)
	section.Key("mumuWindowWidth").SetValue(fmt.Sprintf("%d", config.MuMuWindowWidth))
	section.Key("mumuWindowHeight").SetValue(fmt.Sprintf("%d", config.MuMuWindowHeight))
	section.Key("logLevel").SetValue(config.LogLevel)
	section.Key("loggingEnabled").SetValue(fmt.Sprintf("%t", config.LoggingEnabled))

	// Save instance-specific settings
	instanceSection := cfg.Section(fmt.Sprintf("Instance%d", config.Instance))
	instanceSection.Key("DeadCheck").SetValue(fmt.Sprintf("%t", config.DeadCheck))

	return cfg.SaveTo(path)
}
