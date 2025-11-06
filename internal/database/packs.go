package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Pack and card tracking operations

// LogPackOpening creates a new pack result entry and returns its ID
func (db *DB) LogPackOpening(
	accountID int,
	activityLogID *int,
	packType string,
	packName *string,
	isGodPack bool,
	cardCount int,
	rarityBreakdown map[string]int,
	packPointsEarned int,
) (int64, error) {
	var packID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		// Serialize rarity breakdown to JSON
		var rarityJSON *string
		if rarityBreakdown != nil {
			jsonBytes, err := json.Marshal(rarityBreakdown)
			if err != nil {
				return fmt.Errorf("failed to marshal rarity breakdown: %w", err)
			}
			jsonStr := string(jsonBytes)
			rarityJSON = &jsonStr
		}

		result, err := tx.Exec(`
			INSERT INTO pack_results (
				account_id, activity_log_id, pack_type, pack_name,
				is_god_pack, card_count, rarity_breakdown,
				pack_points_earned, opened_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, activityLogID, packType, packName,
			isGodPack, cardCount, rarityJSON,
			packPointsEarned, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert pack result: %w", err)
		}

		packID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		return 0, err
	}

	return packID, nil
}

// LogCardPulled adds a card to a pack result
func (db *DB) LogCardPulled(
	packResultID int64,
	accountID int,
	cardID string,
	cardName *string,
	cardNumber *string,
	rarity string,
	cardType *string,
	isFullArt bool,
	isEx bool,
	detectionConfidence *float64,
) (int64, error) {
	var cardPulledID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			INSERT INTO cards_pulled (
				pack_result_id, account_id, card_id, card_name,
				card_number, rarity, card_type, is_full_art,
				is_ex, detection_confidence, detected_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, packResultID, accountID, cardID, cardName,
			cardNumber, rarity, cardType, isFullArt,
			isEx, detectionConfidence, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert card pulled: %w", err)
		}

		cardPulledID, err = result.LastInsertId()
		if err != nil {
			return err
		}

		// Update account collection
		err = db.updateAccountCollectionTx(tx, accountID, cardID, cardName, cardNumber, rarity)
		return err
	})

	if err != nil {
		return 0, err
	}

	return cardPulledID, nil
}

// updateAccountCollectionTx updates or inserts a card in the account's collection
func (db *DB) updateAccountCollectionTx(
	tx *sql.Tx,
	accountID int,
	cardID string,
	cardName *string,
	cardNumber *string,
	rarity string,
) error {
	now := time.Now()

	// Check if card already exists
	var existingID int
	var quantity int
	err := tx.QueryRow(`
		SELECT id, quantity
		FROM account_collection
		WHERE account_id = ? AND card_id = ?
	`, accountID, cardID).Scan(&existingID, &quantity)

	if err == sql.ErrNoRows {
		// Insert new card
		name := ""
		if cardName != nil {
			name = *cardName
		}

		_, err = tx.Exec(`
			INSERT INTO account_collection (
				account_id, card_id, card_name, card_number,
				rarity, quantity, first_obtained_at, last_obtained_at
			) VALUES (?, ?, ?, ?, ?, 1, ?, ?)
		`, accountID, cardID, name, cardNumber, rarity, now, now)
		return err
	} else if err != nil {
		return err
	}

	// Update existing card
	_, err = tx.Exec(`
		UPDATE account_collection
		SET quantity = quantity + 1,
			last_obtained_at = ?
		WHERE id = ?
	`, now, existingID)

	return err
}

// GetPackResultByID retrieves a pack result by ID
func (db *DB) GetPackResultByID(packID int64) (*PackResult, error) {
	pack := &PackResult{}
	err := db.conn.QueryRow(`
		SELECT
			id, account_id, activity_log_id, pack_type, pack_name,
			is_god_pack, card_count, rarity_breakdown,
			pack_points_earned, opened_at
		FROM pack_results
		WHERE id = ?
	`, packID).Scan(
		&pack.ID, &pack.AccountID, &pack.ActivityLogID,
		&pack.PackType, &pack.PackName, &pack.IsGodPack,
		&pack.CardCount, &pack.RarityBreakdown,
		&pack.PackPointsEarned, &pack.OpenedAt,
	)

	if err != nil {
		return nil, err
	}

	return pack, nil
}

// GetCardsFromPack retrieves all cards pulled from a specific pack
func (db *DB) GetCardsFromPack(packResultID int64) ([]*CardPulled, error) {
	rows, err := db.conn.Query(`
		SELECT
			id, pack_result_id, account_id, card_id, card_name,
			card_number, rarity, card_type, is_full_art, is_ex,
			detection_confidence, detected_at
		FROM cards_pulled
		WHERE pack_result_id = ?
		ORDER BY detected_at ASC
	`, packResultID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cards := []*CardPulled{}
	for rows.Next() {
		card := &CardPulled{}
		err := rows.Scan(
			&card.ID, &card.PackResultID, &card.AccountID,
			&card.CardID, &card.CardName, &card.CardNumber,
			&card.Rarity, &card.CardType, &card.IsFullArt,
			&card.IsEx, &card.DetectionConfidence, &card.DetectedAt,
		)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}

	return cards, rows.Err()
}

// GetRecentPacksForAccount returns recent pack openings for an account
func (db *DB) GetRecentPacksForAccount(accountID int, limit int) ([]*PackResult, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, pack_type, pack_name,
			is_god_pack, card_count, rarity_breakdown,
			pack_points_earned, opened_at
		FROM pack_results
		WHERE account_id = ?
		ORDER BY opened_at DESC
		LIMIT ?
	`, accountID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packs := []*PackResult{}
	for rows.Next() {
		pack := &PackResult{}
		err := rows.Scan(
			&pack.ID, &pack.AccountID, &pack.ActivityLogID,
			&pack.PackType, &pack.PackName, &pack.IsGodPack,
			&pack.CardCount, &pack.RarityBreakdown,
			&pack.PackPointsEarned, &pack.OpenedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}

	return packs, rows.Err()
}

// GetGodPacksForAccount returns all god packs for an account
func (db *DB) GetGodPacksForAccount(accountID int) ([]*PackResult, error) {
	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, pack_type, pack_name,
			is_god_pack, card_count, rarity_breakdown,
			pack_points_earned, opened_at
		FROM pack_results
		WHERE account_id = ? AND is_god_pack = 1
		ORDER BY opened_at DESC
	`, accountID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	packs := []*PackResult{}
	for rows.Next() {
		pack := &PackResult{}
		err := rows.Scan(
			&pack.ID, &pack.AccountID, &pack.ActivityLogID,
			&pack.PackType, &pack.PackName, &pack.IsGodPack,
			&pack.CardCount, &pack.RarityBreakdown,
			&pack.PackPointsEarned, &pack.OpenedAt,
		)
		if err != nil {
			return nil, err
		}
		packs = append(packs, pack)
	}

	return packs, rows.Err()
}

// GetAccountCollection returns all cards owned by an account
func (db *DB) GetAccountCollection(accountID int) ([]*AccountCollection, error) {
	rows, err := db.conn.Query(`
		SELECT
			id, account_id, card_id, card_name, card_number,
			rarity, quantity, first_obtained_at, last_obtained_at
		FROM account_collection
		WHERE account_id = ?
		ORDER BY rarity DESC, card_name ASC
	`, accountID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collection := []*AccountCollection{}
	for rows.Next() {
		item := &AccountCollection{}
		err := rows.Scan(
			&item.ID, &item.AccountID, &item.CardID,
			&item.CardName, &item.CardNumber, &item.Rarity,
			&item.Quantity, &item.FirstObtainedAt, &item.LastObtainedAt,
		)
		if err != nil {
			return nil, err
		}
		collection = append(collection, item)
	}

	return collection, rows.Err()
}

// GetPackStatistics returns pack statistics from the view
func (db *DB) GetPackStatistics(accountID int) (*PackStatistics, error) {
	stats := &PackStatistics{}
	err := db.conn.QueryRow(`
		SELECT
			account_id, username, total_packs_opened,
			god_packs, pack_types_opened, last_pack_opened
		FROM v_pack_statistics
		WHERE account_id = ?
	`, accountID).Scan(
		&stats.AccountID, &stats.Username, &stats.TotalPacksOpened,
		&stats.GodPacks, &stats.PackTypesOpened, &stats.LastPackOpened,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetRarityDistribution returns the count of cards by rarity for an account
func (db *DB) GetRarityDistribution(accountID int) (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT rarity, SUM(quantity) as count
		FROM account_collection
		WHERE account_id = ?
		GROUP BY rarity
	`, accountID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := make(map[string]int)
	for rows.Next() {
		var rarity string
		var count int
		if err := rows.Scan(&rarity, &count); err != nil {
			return nil, err
		}
		distribution[rarity] = count
	}

	return distribution, rows.Err()
}

// LogWonderPick creates a wonder pick result entry
func (db *DB) LogWonderPick(
	accountID int,
	activityLogID *int,
	cardSelected *string,
	cardRarity *string,
	success bool,
	energyCost int,
	wasFree bool,
) (int64, error) {
	var wonderPickID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			INSERT INTO wonder_pick_results (
				account_id, activity_log_id, card_selected,
				card_rarity, success, energy_cost, was_free, picked_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, activityLogID, cardSelected,
			cardRarity, success, energyCost, wasFree, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert wonder pick result: %w", err)
		}

		wonderPickID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		return 0, err
	}

	return wonderPickID, nil
}

// GetRecentWonderPicks returns recent wonder picks for an account
func (db *DB) GetRecentWonderPicks(accountID int, limit int) ([]*WonderPickResult, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, card_selected,
			card_rarity, success, energy_cost, was_free, picked_at
		FROM wonder_pick_results
		WHERE account_id = ?
		ORDER BY picked_at DESC
		LIMIT ?
	`, accountID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	picks := []*WonderPickResult{}
	for rows.Next() {
		pick := &WonderPickResult{}
		err := rows.Scan(
			&pick.ID, &pick.AccountID, &pick.ActivityLogID,
			&pick.CardSelected, &pick.CardRarity, &pick.Success,
			&pick.EnergyCost, &pick.WasFree, &pick.PickedAt,
		)
		if err != nil {
			return nil, err
		}
		picks = append(picks, pick)
	}

	return picks, rows.Err()
}
