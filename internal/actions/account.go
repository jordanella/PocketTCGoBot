package actions

import (
	"context"
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
)

// InjectNextAccount requests the next available account from the pool and injects it
type InjectNextAccount struct {
	Timeout      int    `yaml:"timeout"`        // Timeout in milliseconds (default: 30000)
	SaveResult   string `yaml:"save_result"`    // Variable name to store account ID
	OnNoAccounts string `yaml:"on_no_accounts"` // Action if pool empty: "wait", "stop", "continue" (default: "stop")
}

func (a *InjectNextAccount) Validate(ab *ActionBuilder) error {
	// Validate OnNoAccounts
	if a.OnNoAccounts != "" && a.OnNoAccounts != "wait" && a.OnNoAccounts != "stop" && a.OnNoAccounts != "continue" {
		return fmt.Errorf("on_no_accounts must be 'wait', 'stop', or 'continue', got '%s'", a.OnNoAccounts)
	}

	// Set defaults
	if a.Timeout == 0 {
		a.Timeout = 30000 // 30 seconds default
	}
	if a.OnNoAccounts == "" {
		a.OnNoAccounts = "stop" // Stop by default
	}

	return nil
}

func (a *InjectNextAccount) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "InjectNextAccount",
		execute: func(botIf BotInterface) error {
			// Get account pool from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access account pool")
			}

			// Manager interface now returns accountpool.AccountPool directly
			pool, ok := managerIf.(interface{ AccountPool() accountpool.AccountPool })
			if !ok {
				return fmt.Errorf("bot manager does not provide AccountPool method")
			}

			accountPool := pool.AccountPool()
			if accountPool == nil {
				return fmt.Errorf("no account pool configured in manager")
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(botIf.Context(), time.Duration(a.Timeout)*time.Millisecond)
			defer cancel()

			// Request next account
			account, err := accountPool.GetNext(ctx)
			if err != nil {
				// Handle no accounts available
				if err.Error() == "no accounts available" || err.Error() == "account pool is closed" {
					switch a.OnNoAccounts {
					case "wait":
						// Already waited via GetNext with timeout
						return fmt.Errorf("timeout waiting for accounts: %w", err)
					case "stop":
						return fmt.Errorf("no accounts available, stopping: %w", err)
					case "continue":
						fmt.Printf("Bot %d: No accounts available, continuing without injection\n", botIf.Instance())
						return nil
					}
				}
				return fmt.Errorf("failed to get next account: %w", err)
			}

			// Update account to track which bot is using it
			account.AssignedTo = botIf.Instance()

			// Inject the account
			if err := botIf.InjectAccount(account); err != nil {
				// Return account to pool on injection failure
				accountPool.Return(account)
				return fmt.Errorf("failed to inject account: %w", err)
			}

			// Save account ID to variable if requested
			if a.SaveResult != "" {
				botIf.Variables().Set(a.SaveResult, account.ID)
				fmt.Printf("Bot %d: Stored account ID '%s' in variable '%s'\n", botIf.Instance(), account.ID, a.SaveResult)
			}

			fmt.Printf("Bot %d: Account '%s' assigned and injected\n", botIf.Instance(), account.ID)
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// CompleteAccount marks the current account as successfully processed
type CompleteAccount struct {
	AccountID   string `yaml:"account_id"`    // Variable containing account ID (default: uses current account)
	Success     bool   `yaml:"success"`       // Whether processing was successful (default: true)
	PacksOpened int    `yaml:"packs_opened"`  // Number of packs opened
	CardsFound  int    `yaml:"cards_found"`   // Number of cards found
	StarsTotal  int    `yaml:"stars_total"`   // Total stars collected
	KeepCount   int    `yaml:"keep_count"`    // Number of cards kept
	Error       string `yaml:"error"`         // Error message if failed
}

func (a *CompleteAccount) Validate(ab *ActionBuilder) error {
	// Set default success to true
	if !a.Success && a.Error == "" {
		// If not successful, should have an error message
		return fmt.Errorf("if success=false, must provide an error message")
	}
	return nil
}

func (a *CompleteAccount) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "CompleteAccount",
		execute: func(botIf BotInterface) error {
			// Get account pool from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access account pool")
			}

			// Manager interface now returns accountpool.AccountPool directly
			pool, ok := managerIf.(interface{ AccountPool() accountpool.AccountPool })
			if !ok {
				return fmt.Errorf("bot manager does not provide AccountPool method")
			}

			accountPool := pool.AccountPool()
			if accountPool == nil {
				return fmt.Errorf("no account pool configured in manager")
			}

			// Get account to mark complete
			var account *accountpool.Account
			if a.AccountID != "" {
				// Get account ID from variable
				accountID, exists := botIf.Variables().Get(a.AccountID)
				if !exists || accountID == "" {
					return fmt.Errorf("variable '%s' is empty or not set", a.AccountID)
				}

				// Retrieve account from pool
				var err error
				account, err = accountPool.GetByID(accountID)
				if err != nil {
					return fmt.Errorf("failed to get account '%s': %w", accountID, err)
				}
			} else {
				// Use current account
				accountIf := botIf.GetCurrentAccount()
				if accountIf == nil {
					return fmt.Errorf("no current account assigned to bot")
				}

				// Type assert to concrete Account
				var ok bool
				account, ok = accountIf.(*accountpool.Account)
				if !ok {
					return fmt.Errorf("current account is not a *accountpool.Account")
				}
			}

			// Create result
			result := accountpool.AccountResult{
				Success:     a.Success,
				PacksOpened: a.PacksOpened,
				CardsFound:  a.CardsFound,
				StarsTotal:  a.StarsTotal,
				KeepCount:   a.KeepCount,
				Error:       a.Error,
				Timestamp:   time.Now(),
				BotInstance: botIf.Instance(),
			}

			// Calculate duration if account has AssignedAt time
			if account.AssignedAt != nil {
				result.Duration = time.Since(*account.AssignedAt)
			}

			// Mark account as used
			if err := accountPool.MarkUsed(account, result); err != nil {
				return fmt.Errorf("failed to mark account complete: %w", err)
			}

			// Clear current account from bot
			botIf.ClearCurrentAccount()

			fmt.Printf("Bot %d: Account '%s' marked as %s\n", botIf.Instance(), account.ID,
				map[bool]string{true: "completed", false: "failed"}[a.Success])

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// ReturnAccount returns an account back to the pool without marking it complete
// Useful when an account needs to be re-queued (e.g., due to temporary error)
type ReturnAccount struct {
	AccountID string `yaml:"account_id"` // Variable containing account ID (default: uses current account)
	Reason    string `yaml:"reason"`     // Optional reason for returning
}

func (a *ReturnAccount) Validate(ab *ActionBuilder) error {
	return nil
}

func (a *ReturnAccount) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "ReturnAccount",
		execute: func(botIf BotInterface) error {
			// Get account pool from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access account pool")
			}

			// Manager interface now returns accountpool.AccountPool directly
			pool, ok := managerIf.(interface{ AccountPool() accountpool.AccountPool })
			if !ok {
				return fmt.Errorf("bot manager does not provide AccountPool method")
			}

			accountPool := pool.AccountPool()
			if accountPool == nil {
				return fmt.Errorf("no account pool configured in manager")
			}

			// Get account to return
			var account *accountpool.Account
			if a.AccountID != "" {
				// Get account ID from variable
				accountID, exists := botIf.Variables().Get(a.AccountID)
				if !exists || accountID == "" {
					return fmt.Errorf("variable '%s' is empty or not set", a.AccountID)
				}

				// Retrieve account from pool
				var err error
				account, err = accountPool.GetByID(accountID)
				if err != nil {
					return fmt.Errorf("failed to get account '%s': %w", accountID, err)
				}
			} else {
				// Use current account
				accountIf := botIf.GetCurrentAccount()
				if accountIf == nil {
					return fmt.Errorf("no current account assigned to bot")
				}

				// Type assert to concrete Account
				var ok bool
				account, ok = accountIf.(*accountpool.Account)
				if !ok {
					return fmt.Errorf("current account is not a *accountpool.Account")
				}
			}

			// Return account to pool
			if err := accountPool.Return(account); err != nil {
				return fmt.Errorf("failed to return account: %w", err)
			}

			// Clear current account from bot
			botIf.ClearCurrentAccount()

			if a.Reason != "" {
				fmt.Printf("Bot %d: Account '%s' returned to pool (%s)\n", botIf.Instance(), account.ID, a.Reason)
			} else {
				fmt.Printf("Bot %d: Account '%s' returned to pool\n", botIf.Instance(), account.ID)
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// MarkAccountFailed marks an account as failed
type MarkAccountFailed struct {
	AccountID string `yaml:"account_id"` // Variable containing account ID (default: uses current account)
	Reason    string `yaml:"reason"`     // Reason for failure (required)
}

func (a *MarkAccountFailed) Validate(ab *ActionBuilder) error {
	if a.Reason == "" {
		return fmt.Errorf("reason is required for MarkAccountFailed")
	}
	return nil
}

func (a *MarkAccountFailed) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "MarkAccountFailed",
		execute: func(botIf BotInterface) error {
			// Get account pool from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access account pool")
			}

			// Manager interface now returns accountpool.AccountPool directly
			pool, ok := managerIf.(interface{ AccountPool() accountpool.AccountPool })
			if !ok {
				return fmt.Errorf("bot manager does not provide AccountPool method")
			}

			accountPool := pool.AccountPool()
			if accountPool == nil {
				return fmt.Errorf("no account pool configured in manager")
			}

			// Get account to mark failed
			var account *accountpool.Account
			if a.AccountID != "" {
				// Get account ID from variable
				accountID, exists := botIf.Variables().Get(a.AccountID)
				if !exists || accountID == "" {
					return fmt.Errorf("variable '%s' is empty or not set", a.AccountID)
				}

				// Retrieve account from pool
				var err error
				account, err = accountPool.GetByID(accountID)
				if err != nil {
					return fmt.Errorf("failed to get account '%s': %w", accountID, err)
				}
			} else {
				// Use current account
				accountIf := botIf.GetCurrentAccount()
				if accountIf == nil {
					return fmt.Errorf("no current account assigned to bot")
				}

				// Type assert to concrete Account
				var ok bool
				account, ok = accountIf.(*accountpool.Account)
				if !ok {
					return fmt.Errorf("current account is not a *accountpool.Account")
				}
			}

			// Mark account as failed
			if err := accountPool.MarkFailed(account, a.Reason); err != nil {
				return fmt.Errorf("failed to mark account as failed: %w", err)
			}

			// Clear current account from bot
			botIf.ClearCurrentAccount()

			fmt.Printf("Bot %d: Account '%s' marked as failed: %s\n", botIf.Instance(), account.ID, a.Reason)

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
