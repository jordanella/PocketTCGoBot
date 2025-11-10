package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/pkg/templates"

	_ "github.com/mattn/go-sqlite3"
)

// Example: Using the Orchestrator with PoolManager for SQL-based account pools
// This demonstrates the complete flow from pool discovery to bot group launch
func main() {
	fmt.Println("=== Orchestrator with SQL Account Pools Example ===\n")

	// ===== Step 1: Initialize Core Components =====

	// Create bot configuration
	config := &bot.Config{
		TemplatesDir: "./templates",
		RoutinesDir:  "./routines",
		EmulatorPath: `C:\Program Files\Netease\MuMuPlayer-12.0\shell\MuMuPlayer.exe`,
	}

	// Create template registry
	templateRegistry := templates.NewTemplateRegistry(config.TemplatesDir)
	if err := templateRegistry.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
	fmt.Printf("✓ Loaded %d templates\n", len(templateRegistry.ListTemplates()))

	// Create routine registry
	routineRegistry := actions.NewRoutineRegistry(config.RoutinesDir)
	if err := routineRegistry.LoadRoutines(); err != nil {
		log.Fatalf("Failed to load routines: %v", err)
	}
	fmt.Printf("✓ Loaded %d routines\n", len(routineRegistry.ListRoutines()))

	// Create emulator manager
	emulatorManager := emulator.NewManager(config.EmulatorPath)
	// Register some instances
	emulatorManager.RegisterInstance(0, 16384)
	emulatorManager.RegisterInstance(1, 16416)
	emulatorManager.RegisterInstance(2, 16448)
	fmt.Printf("✓ Registered 3 emulator instances\n")

	// Open database
	db, err := sql.Open("sqlite3", "./accounts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	fmt.Printf("✓ Connected to database\n")

	// ===== Step 2: Initialize PoolManager =====

	poolManager := accountpool.NewPoolManager("./pools", db, "account_xmls")

	// Discover pools from YAML files in ./pools directory
	if err := poolManager.DiscoverPools(); err != nil {
		log.Fatalf("Failed to discover pools: %v", err)
	}

	pools := poolManager.ListPools()
	fmt.Printf("\n✓ Discovered %d account pools:\n", len(pools))
	for _, poolName := range pools {
		poolDef, _ := poolManager.GetPoolDefinition(poolName)
		fmt.Printf("  - %s (%s type)\n", poolName, poolDef.Type)
	}

	// ===== Step 3: Create Orchestrator =====

	orchestrator := bot.NewOrchestrator(
		config,
		templateRegistry,
		routineRegistry,
		emulatorManager,
		poolManager,
	)

	// Set stagger delay between bot launches
	orchestrator.SetStaggerDelay(3 * time.Second)
	fmt.Printf("\n✓ Created orchestrator with 3s stagger delay\n")

	// ===== Step 4: Create Bot Group =====

	// Create a group that will use 2 bots on instances 0 and 1
	group, err := orchestrator.CreateGroup(
		"Premium Farmers",           // Group name
		"OpenPacks",                  // Routine to run
		[]int{0, 1, 2},               // Available emulator instances
		2,                            // Requested bot count
		map[string]string{},          // Routine config overrides
		"Premium Farmers Pool",       // Account pool name (from discovered pools)
	)
	if err != nil {
		log.Fatalf("Failed to create group: %v", err)
	}
	fmt.Printf("\n✓ Created bot group '%s'\n", group.Name)
	fmt.Printf("  - Routine: %s\n", group.RoutineName)
	fmt.Printf("  - Bot count: %d\n", group.RequestedBotCount)
	fmt.Printf("  - Pool: %s\n", group.AccountPoolName)

	// ===== Step 5: Test Pool Before Launch (Optional) =====

	fmt.Println("\n=== Testing Account Pool ===")
	testResult, err := poolManager.TestPool("Premium Farmers Pool")
	if err != nil {
		log.Fatalf("Failed to test pool: %v", err)
	}

	if testResult.Success {
		fmt.Printf("✓ Pool test successful\n")
		fmt.Printf("  - Accounts found: %d\n", testResult.AccountsFound)
	} else {
		fmt.Printf("✗ Pool test failed: %s\n", testResult.Error)
		return
	}

	// ===== Step 6: Refresh Pool Before Launch (User Prompted in GUI) =====

	fmt.Println("\n=== Refreshing Account Pool ===")
	if err := orchestrator.RefreshGroupAccountPool("Premium Farmers"); err != nil {
		log.Printf("Warning: Failed to refresh pool: %v", err)
	} else {
		fmt.Println("✓ Pool refreshed with latest data")
	}

	// ===== Step 7: Launch Bot Group =====

	fmt.Println("\n=== Launching Bot Group ===")

	launchOptions := bot.LaunchOptions{
		ValidateRoutine:   true,
		ValidateTemplates: true,
		ValidateEmulators: true,
		OnConflict:        bot.ConflictResolutionSkip, // Skip conflicting instances
		StaggerDelay:      3 * time.Second,
		EmulatorTimeout:   60 * time.Second,
		RestartPolicy:     bot.RestartPolicyNever,
	}

	result, err := orchestrator.LaunchGroup("Premium Farmers", launchOptions)
	if err != nil {
		log.Printf("Launch completed with errors: %v", err)
	}

	// Display launch results
	fmt.Printf("\n=== Launch Results ===\n")
	fmt.Printf("Success: %v\n", result.Success)
	fmt.Printf("Launched: %d / %d requested\n", result.LaunchedBots, result.RequestedBots)

	if len(result.Conflicts) > 0 {
		fmt.Printf("\nConflicts detected:\n")
		for _, conflict := range result.Conflicts {
			fmt.Printf("  - Instance %d in use by group '%s'\n",
				conflict.InstanceID, conflict.ConflictingGroup)
		}
	}

	if len(result.SkippedInstances) > 0 {
		fmt.Printf("\nSkipped instances: %v\n", result.SkippedInstances)
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}

	// ===== Step 8: Monitor Bots =====

	if result.LaunchedBots > 0 {
		fmt.Println("\n=== Monitoring Bots ===")

		// Wait a bit for bots to start
		time.Sleep(2 * time.Second)

		// Get bot status
		activeBots := group.GetAllBotInfo()
		fmt.Printf("\nActive bots: %d\n", len(activeBots))
		for instanceID, botInfo := range activeBots {
			fmt.Printf("  Instance %d: %s (started %s ago)\n",
				instanceID,
				botInfo.Status,
				time.Since(botInfo.StartedAt).Round(time.Second))
		}

		// Get pool stats
		if group.AccountPool != nil {
			stats := group.AccountPool.GetStats()
			fmt.Printf("\nAccount Pool Stats:\n")
			fmt.Printf("  Total: %d\n", stats.Total)
			fmt.Printf("  Available: %d\n", stats.Available)
			fmt.Printf("  In Use: %d\n", stats.InUse)
			fmt.Printf("  Completed: %d\n", stats.Completed)
			fmt.Printf("  Failed: %d\n", stats.Failed)
		}

		// In a real application, you would monitor bots until completion
		// For this example, we'll just wait briefly then stop
		fmt.Println("\n(In production, bots would continue running...)")
		fmt.Println("Waiting 10 seconds before stopping...")
		time.Sleep(10 * time.Second)

		// ===== Step 9: Stop Bot Group =====

		fmt.Println("\n=== Stopping Bot Group ===")
		if err := orchestrator.StopGroup("Premium Farmers"); err != nil {
			log.Printf("Error stopping group: %v", err)
		} else {
			fmt.Println("✓ Group stopped successfully")
		}
	}

	// ===== Step 10: Cleanup =====

	fmt.Println("\n=== Cleanup ===")

	// Delete group
	if err := orchestrator.DeleteGroup("Premium Farmers"); err != nil {
		log.Printf("Error deleting group: %v", err)
	} else {
		fmt.Println("✓ Group deleted")
	}

	// Close all pool instances
	if err := poolManager.CloseAll(); err != nil {
		log.Printf("Error closing pools: %v", err)
	} else {
		fmt.Println("✓ All pools closed")
	}

	fmt.Println("\n=== Example Complete ===")
}
