package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// Example showing how to set up multiple manager "groups"
// Each manager has its own config and account pool, but shares global registries
func main() {
	// ============================================================
	// STEP 1: Create GLOBAL registries (shared by all managers)
	// ============================================================
	fmt.Println("=== Loading Global Registries ===")

	// Load templates (shared by all bots)
	templatesPath := filepath.Join(".", "templates")
	templateRegistry := templates.NewTemplateRegistry(templatesPath)
	if err := templateRegistry.LoadFromDirectory(filepath.Join(templatesPath, "registry")); err != nil {
		fmt.Printf("Warning: Template loading failed: %v\n", err)
	}
	fmt.Printf("Loaded templates from: %s\n", templatesPath)

	// Load routines (shared by all bots)
	routinesPath := filepath.Join(".", "routines")
	routineRegistry := actions.NewRoutineRegistry(routinesPath).WithTemplateRegistry(templateRegistry)
	fmt.Printf("Loaded routines from: %s\n", routinesPath)

	// ============================================================
	// STEP 2: Create Manager Groups (each with different config/pool)
	// ============================================================

	// --- GROUP A: Premium Account Farmers ---
	configA := loadConfig()
	configA.Instance = 0 // Will be overridden per bot
	configA.VerboseLogging = true

	managerA := bot.NewManagerWithRegistries(configA, templateRegistry, routineRegistry)

	// Setup account pool for Group A
	poolA, err := accountpool.NewFileAccountPool("./accounts/premium", accountpool.PoolConfig{
		MinPacks:        10,
		SortMethod:      accountpool.SortMethodPacksDesc,
		RetryFailed:     true,
		MaxFailures:     3,
		WaitForAccounts: true,
		MaxWaitTime:     5 * time.Minute,
		BufferSize:      100,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create pool A: %v", err))
	}
	managerA.SetAccountPool(poolA)

	fmt.Println("\n=== Group A: Premium Farmers ===")
	fmt.Println("  Config: Premium settings")
	fmt.Println("  Accounts: ./accounts/premium (min 10 packs)")
	fmt.Println("  Routine: farm_premium_packs")
	fmt.Println("  Bots: 1-4")

	// Create bots for Group A
	for i := 1; i <= 4; i++ {
		botA, err := managerA.CreateBot(i)
		if err != nil {
			panic(fmt.Sprintf("Failed to create bot %d: %v", i, err))
		}

		// Start routine in background
		go func(b *bot.Bot, instance int) {
			fmt.Printf("Bot %d (Group A): Starting farm_premium_packs\n", instance)
			policy := bot.RestartPolicy{
				Enabled:        true,
				MaxRetries:     5,
				InitialDelay:   10 * time.Second,
				MaxDelay:       5 * time.Minute,
				BackoffFactor:  2.0,
				ResetOnSuccess: true,
			}
			if err := managerA.ExecuteWithRestart(instance, "farm_premium_packs", policy); err != nil {
				fmt.Printf("Bot %d (Group A): Failed - %v\n", instance, err)
			}
		}(botA, i)
	}

	// --- GROUP B: Mission Runners ---
	configB := loadConfig()
	configB.Instance = 0
	configB.VerboseLogging = false // Less verbose for this group
	configB.Delay = 150            // Faster delays for missions

	managerB := bot.NewManagerWithRegistries(configB, templateRegistry, routineRegistry)

	// Setup account pool for Group B
	poolB, err := accountpool.NewFileAccountPool("./accounts/standard", accountpool.PoolConfig{
		MinPacks:        5,
		SortMethod:      accountpool.SortMethodModifiedAsc,
		RetryFailed:     false,
		WaitForAccounts: true,
		MaxWaitTime:     2 * time.Minute,
		BufferSize:      50,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create pool B: %v", err))
	}
	managerB.SetAccountPool(poolB)

	fmt.Println("\n=== Group B: Mission Runners ===")
	fmt.Println("  Config: Fast settings (delay=150ms)")
	fmt.Println("  Accounts: ./accounts/standard (min 5 packs)")
	fmt.Println("  Routine: complete_daily_missions")
	fmt.Println("  Bots: 5-6")

	// Create bots for Group B
	for i := 5; i <= 6; i++ {
		botB, err := managerB.CreateBot(i)
		if err != nil {
			panic(fmt.Sprintf("Failed to create bot %d: %v", i, err))
		}

		go func(b *bot.Bot, instance int) {
			fmt.Printf("Bot %d (Group B): Starting complete_daily_missions\n", instance)
			policy := bot.RestartPolicy{Enabled: false} // No retries for missions
			if err := managerB.ExecuteWithRestart(instance, "complete_daily_missions", policy); err != nil {
				fmt.Printf("Bot %d (Group B): Failed - %v\n", instance, err)
			}
		}(botB, i)
	}

	// --- GROUP C: Single Test Bot (no account pool) ---
	configC := loadConfig()
	configC.Instance = 7
	configC.VerboseLogging = true

	managerC := bot.NewManagerWithRegistries(configC, templateRegistry, routineRegistry)
	// Note: No account pool set - manual injection or routine doesn't use accounts

	fmt.Println("\n=== Group C: Test Bot ===")
	fmt.Println("  Config: Test settings")
	fmt.Println("  Accounts: None (manual injection)")
	fmt.Println("  Routine: test_new_feature")
	fmt.Println("  Bots: 7")

	botC, err := managerC.CreateBot(7)
	if err != nil {
		panic(fmt.Sprintf("Failed to create bot 7: %v", err))
	}

	go func(b *bot.Bot) {
		fmt.Printf("Bot 7 (Group C): Starting test_new_feature\n")
		policy := bot.RestartPolicy{Enabled: false}
		if err := managerC.ExecuteWithRestart(7, "test_new_feature", policy); err != nil {
			fmt.Printf("Bot 7 (Group C): Failed - %v\n", err)
		}
	}(botC)

	// ============================================================
	// STEP 3: Monitor stats
	// ============================================================
	go monitorStats(managerA, managerB, managerC)

	// ============================================================
	// STEP 4: Wait for shutdown
	// ============================================================
	fmt.Println("\n=== All Groups Running - Press Ctrl+C to stop ===")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n=== Shutting down ===")
	managerA.ShutdownAll()
	managerB.ShutdownAll()
	managerC.ShutdownAll()

	if poolA != nil {
		poolA.Close()
	}
	if poolB != nil {
		poolB.Close()
	}

	fmt.Println("=== Shutdown complete ===")
}

func monitorStats(managers ...*bot.Manager) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n=== Manager Statistics ===")

		for i, manager := range managers {
			groupName := []string{"Group A (Premium)", "Group B (Missions)", "Group C (Test)"}[i]
			fmt.Printf("\n%s:\n", groupName)

			pool := manager.AccountPool()
			if pool != nil {
				stats := pool.GetStats()
				fmt.Printf("  Pool: %d total | %d available | %d in use | %d completed | %d failed\n",
					stats.Total, stats.Available, stats.InUse, stats.Completed, stats.Failed)
				fmt.Printf("  Results: %d packs | %d cards | %d stars | %d keeps\n",
					stats.TotalPacksOpened, stats.TotalCardsFound, stats.TotalStars, stats.TotalKeeps)
			} else {
				fmt.Println("  Pool: None configured")
			}
		}
		fmt.Println("==========================")
	}
}

func loadConfig() *bot.Config {
	config := &bot.Config{
		FolderPath:       "C:\\Program Files\\MuMuPlayer-12.0",
		DefaultLanguage:  "Scale100",
		Delay:            250,
		SwipeSpeed:       500,
		WaitTime:         5,
		ShowStatus:       true,
		MuMuWindowWidth:  540,
		MuMuWindowHeight: 960,
	}
	config.ApplyDefaults()
	return config
}
