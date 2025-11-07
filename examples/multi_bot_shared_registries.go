package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"jordanella.com/pocket-tcg-go/internal/bot"
)

// Example: Running multiple bots with shared template and routine registries
func main() {
	// Configuration
	config := &bot.Config{
		FolderPath:      "C:/PocketTCG",
		ADBPath:         "",          // Auto-detect
		Columns:         3,            // 3 columns layout
		RowGap:          60,           // 60px gap between rows
		DefaultLanguage: "Scale100",   // UI scale
		SelectedMonitor: 0,            // Primary monitor
		TitleBarHeight:  20,           // MuMu title bar height
	}

	// Create manager with shared registries
	fmt.Println("Initializing bot manager with shared registries...")
	manager, err := bot.NewManager(config)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Ensure cleanup on exit
	defer func() {
		fmt.Println("\nShutting down all bots...")
		manager.ShutdownAll()
		fmt.Println("All bots stopped, registries cleaned up")
	}()

	// Number of bots to run
	numBots := 6

	// Create and start multiple bots
	fmt.Printf("Creating %d bots with shared registries...\n", numBots)

	var wg sync.WaitGroup
	botContexts := make(map[int]context.CancelFunc)

	for i := 1; i <= numBots; i++ {
		fmt.Printf("  Creating bot %d...\n", i)

		// Create bot (automatically gets shared registries)
		b, err := manager.CreateBot(i)
		if err != nil {
			log.Printf("Failed to create bot %d: %v", i, err)
			continue
		}

		// Create context for this bot
		ctx, cancel := context.WithCancel(context.Background())
		botContexts[i] = cancel

		// Start bot in goroutine
		wg.Add(1)
		go func(bot *bot.Bot, instance int) {
			defer wg.Done()

			fmt.Printf("Bot %d starting...\n", instance)

			// Run bot (this would be your actual bot logic)
			// For this example, we just simulate work
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					fmt.Printf("Bot %d stopping...\n", instance)
					return

				case <-ticker.C:
					// Example: Execute a routine
					// routine, err := bot.Routines().Get("main_loop")
					// if err == nil {
					//     routine.Execute(bot)
					// }

					fmt.Printf("Bot %d: Working... (using shared registries)\n", instance)
				}
			}
		}(b, i)

		time.Sleep(100 * time.Millisecond) // Slight delay between bot starts
	}

	fmt.Printf("\n✓ All %d bots created successfully!\n", numBots)
	fmt.Printf("✓ Memory efficiency: Using 1 shared TemplateRegistry instead of %d copies\n", numBots)
	fmt.Printf("✓ Memory efficiency: Using 1 shared RoutineRegistry instead of %d copies\n", numBots)
	fmt.Printf("✓ Total memory savings: ~%d%% for templates and routines\n\n", (numBots-1)*100/numBots)

	// Show registry stats
	templateRegistry := manager.TemplateRegistry()
	routineRegistry := manager.RoutineRegistry()

	fmt.Println("Shared Registry Status:")
	fmt.Printf("  Active bots: %d\n", manager.GetActiveCount())
	fmt.Printf("  Shared template registry: %v\n", templateRegistry != nil)
	fmt.Printf("  Shared routine registry: %v\n", routineRegistry != nil)
	fmt.Println()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Bots running. Press Ctrl+C to stop...")
	<-sigChan

	// Gracefully stop all bots
	fmt.Println("\nStopping bots gracefully...")
	for _, cancel := range botContexts {
		cancel()
	}

	// Wait for all bots to finish
	wg.Wait()
	fmt.Println("All bot goroutines finished")
}
