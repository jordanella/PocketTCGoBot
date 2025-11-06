package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	"jordanella.com/pocket-tcg-go/internal/database"
)

func main() {
	// Parse command line flags
	dbPath := flag.String("db", "", "Path to database file (default: ./bot.db)")
	numAccounts := flag.Int("accounts", 3, "Number of test accounts to create")
	numActivities := flag.Int("activities", 10, "Number of activities per account")
	numPacks := flag.Int("packs", 5, "Number of pack openings per account")
	numErrors := flag.Int("errors", 3, "Number of errors per account")
	flag.Parse()

	// Determine database path
	var finalDBPath string
	if *dbPath != "" {
		finalDBPath = *dbPath
	} else {
		finalDBPath = "bot.db"
	}

	log.Printf("Seeding database at: %s", finalDBPath)

	// Open database
	db, err := database.Open(finalDBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations complete")

	// Seed data
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < *numAccounts; i++ {
		log.Printf("Creating account %d/%d", i+1, *numAccounts)
		seedAccount(db, i, *numActivities, *numPacks, *numErrors)
	}

	log.Println("✓ Database seeding complete!")
}

func seedAccount(db *database.DB, index int, numActivities, numPacks, numErrors int) {
	deviceAccount := fmt.Sprintf("test_device_%d", index+1)
	password := fmt.Sprintf("password%d", index+1)
	filePath := filepath.Join("accounts", fmt.Sprintf("account_%d.json", index+1))

	// Create account
	account, err := db.CreateAccount(deviceAccount, password, filePath)
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		return
	}

	// Set username and friend code
	username := fmt.Sprintf("Player_%d", index+1)
	friendCode := fmt.Sprintf("%04d-%04d-%04d", rand.Intn(10000), rand.Intn(10000), rand.Intn(10000))
	account.Username = &username
	account.FriendCode = &friendCode

	// Update account resources
	shinedust := rand.Intn(10000)
	hourglasses := rand.Intn(100)
	pokegold := rand.Intn(1000)
	packPoints := rand.Intn(500)

	err = db.UpdateAccountResources(account.ID, shinedust, hourglasses, pokegold, packPoints)
	if err != nil {
		log.Printf("Failed to update resources: %v", err)
	}

	// Update account level and stats
	level := rand.Intn(30) + 1
	packsOpened := rand.Intn(50)
	wonderPicks := rand.Intn(20)

	_, err = db.Conn().Exec(`
		UPDATE accounts
		SET account_level = ?, packs_opened = ?, wonder_picks_done = ?
		WHERE id = ?
	`, level, packsOpened, wonderPicks, account.ID)
	if err != nil {
		log.Printf("Failed to update account stats: %v", err)
	}

	// Create activities
	seedActivities(db, account.ID, numActivities)

	// Create pack openings
	seedPackOpenings(db, account.ID, numPacks)

	// Create errors
	seedErrors(db, account.ID, numErrors)

	log.Printf("  ✓ Account %d created: %s (Level %d)", account.ID, username, level)
}

func seedActivities(db *database.DB, accountID int, count int) {
	activityTypes := []string{"pack_opening", "wonder_pick", "mission_completion", "battle", "daily_login"}
	routineNames := []string{"OpenPack", "DoWonderPick", "CompleteMission", "DoBattle", "ClaimDailyBonus"}
	statuses := []string{"completed", "completed", "completed", "failed", "running"}

	for i := 0; i < count; i++ {
		typeIndex := rand.Intn(len(activityTypes))
		activityType := activityTypes[typeIndex]
		routineName := routineNames[typeIndex]
		status := statuses[rand.Intn(len(statuses))]

		// Start activity in the past
		startTime := time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour)

		activityID, err := db.StartActivity(accountID, activityType, routineName, "v1.0.0")
		if err != nil {
			log.Printf("Failed to start activity: %v", err)
			continue
		}

		// Backdate the start time
		_, err = db.Conn().Exec("UPDATE activity_log SET started_at = ? WHERE id = ?", startTime, activityID)
		if err != nil {
			log.Printf("Failed to update start time: %v", err)
		}

		// Complete some activities
		if status == "completed" {
			duration := rand.Intn(300) + 5 // 5-305 seconds
			completedAt := startTime.Add(time.Duration(duration) * time.Second)

			_, err = db.Conn().Exec(`
				UPDATE activity_log
				SET completed_at = ?, duration_seconds = ?, status = 'completed'
				WHERE id = ?
			`, completedAt, duration, activityID)
			if err != nil {
				log.Printf("Failed to complete activity: %v", err)
			}
		} else if status == "failed" {
			completedAt := startTime.Add(time.Duration(rand.Intn(60)+5) * time.Second)
			errorMsg := "Activity failed due to unexpected error"

			_, err = db.Conn().Exec(`
				UPDATE activity_log
				SET completed_at = ?, status = 'failed', error_message = ?
				WHERE id = ?
			`, completedAt, errorMsg, activityID)
			if err != nil {
				log.Printf("Failed to mark activity as failed: %v", err)
			}
		}
		// If status is "running", leave it as is
	}
}

func seedPackOpenings(db *database.DB, accountID int, count int) {
	packTypes := []string{"genetic_apex", "mythical_island"}
	packNames := []string{"Genetic Apex", "Mythical Island"}
	cardNames := []string{"Pikachu", "Charizard", "Mewtwo", "Mew", "Articuno", "Zapdos", "Moltres", "Dragonite", "Eevee", "Snorlax"}

	for i := 0; i < count; i++ {
		packIndex := rand.Intn(len(packTypes))
		packType := packTypes[packIndex]
		packName := packNames[packIndex]
		isGodPack := rand.Float32() < 0.05 // 5% chance of god pack

		rarityBreakdown := map[string]int{
			"1_diamond": 3,
			"2_diamond": 1,
			"3_diamond": 1,
		}

		if isGodPack {
			rarityBreakdown = map[string]int{
				"4_diamond": 5,
			}
		}

		packID, err := db.LogPackOpening(
			accountID,
			nil,
			packType,
			&packName,
			isGodPack,
			5,
			rarityBreakdown,
			rand.Intn(10)+1,
		)
		if err != nil {
			log.Printf("Failed to log pack opening: %v", err)
			continue
		}

		// Add cards to the pack
		numCards := 5
		for j := 0; j < numCards; j++ {
			cardName := cardNames[rand.Intn(len(cardNames))]
			cardNumber := fmt.Sprintf("%03d/165", rand.Intn(165)+1)
			cardType := "pokemon"
			rarity := "1_diamond"

			if j == 4 { // Last card is always rare
				rarities := []string{"3_diamond", "4_diamond"}
				rarity = rarities[rand.Intn(len(rarities))]
			} else if j == 3 {
				rarity = "2_diamond"
			}

			confidence := 0.85 + rand.Float64()*0.14 // 0.85-0.99
			isFullArt := rand.Float32() < 0.1       // 10% chance
			isEx := rand.Float32() < 0.05           // 5% chance

			_, err = db.LogCardPulled(
				packID,
				accountID,
				fmt.Sprintf("%s_%s", cardName, cardNumber),
				&cardName,
				&cardNumber,
				rarity,
				&cardType,
				isFullArt,
				isEx,
				&confidence,
			)
			if err != nil {
				log.Printf("Failed to log card: %v", err)
			}
		}
	}
}

func seedErrors(db *database.DB, accountID int, count int) {
	errorTypes := []string{"popup", "stuck", "no_response", "communication", "timeout"}
	severities := []string{"low", "medium", "high", "critical"}
	templates := []string{"error_popup", "maintenance_screen", "connection_lost", "stuck_loading"}
	actions := []string{"ClickButton", "SwipeUp", "TapCard", "WaitForScreen"}

	for i := 0; i < count; i++ {
		errorType := errorTypes[rand.Intn(len(errorTypes))]
		severity := severities[rand.Intn(len(severities))]
		message := fmt.Sprintf("Test error: %s occurred", errorType)

		stackTrace := "at internal/actions/action.go:42\nat internal/bot/bot.go:156"
		screenState := "HomeScreen"
		template := templates[rand.Intn(len(templates))]
		action := actions[rand.Intn(len(actions))]

		errorID, err := db.LogError(
			&accountID,
			nil,
			errorType,
			severity,
			message,
			&stackTrace,
			&screenState,
			&template,
			&action,
		)
		if err != nil {
			log.Printf("Failed to log error: %v", err)
			continue
		}

		// Mark some errors as recovered
		if rand.Float32() < 0.7 { // 70% recovery rate
			recoveryAction := "Dismissed popup and continued"
			recoveryTime := rand.Intn(5000) + 500 // 500-5500ms

			err = db.MarkErrorRecovered(errorID, recoveryAction, recoveryTime)
			if err != nil {
				log.Printf("Failed to mark error as recovered: %v", err)
			}
		}
	}
}
