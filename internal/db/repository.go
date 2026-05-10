package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash`,
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user User
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, pgx.ErrNoRows
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (s *Store) StorePlaidItem(ctx context.Context, item PlaidItem) (PlaidItem, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO plaid_items (
			user_id, item_id, access_token_ciphertext, access_token_nonce, access_token_key_id, access_token_algorithm
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (item_id) DO UPDATE SET
			access_token_ciphertext = EXCLUDED.access_token_ciphertext,
			access_token_nonce = EXCLUDED.access_token_nonce,
			access_token_key_id = EXCLUDED.access_token_key_id,
			access_token_algorithm = EXCLUDED.access_token_algorithm,
			updated_at = now()
		RETURNING id, user_id, item_id, access_token_ciphertext, access_token_nonce,
			access_token_key_id, access_token_algorithm, COALESCE(transactions_cursor, '')`,
		item.UserID, item.ItemID, item.AccessTokenCiphertext, item.AccessTokenNonce, item.AccessTokenKeyID, item.AccessTokenAlgorithm,
	).Scan(&item.ID, &item.UserID, &item.ItemID, &item.AccessTokenCiphertext, &item.AccessTokenNonce, &item.AccessTokenKeyID, &item.AccessTokenAlgorithm, &item.TransactionsCursor)
	if err != nil {
		return PlaidItem{}, fmt.Errorf("store plaid item: %w", err)
	}
	return item, nil
}

func (s *Store) GetPlaidItemByItemID(ctx context.Context, itemID string) (PlaidItem, error) {
	var item PlaidItem
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, item_id, access_token_ciphertext, access_token_nonce,
			access_token_key_id, access_token_algorithm, COALESCE(transactions_cursor, '')
		 FROM plaid_items WHERE item_id = $1`,
		itemID,
	).Scan(&item.ID, &item.UserID, &item.ItemID, &item.AccessTokenCiphertext, &item.AccessTokenNonce, &item.AccessTokenKeyID, &item.AccessTokenAlgorithm, &item.TransactionsCursor)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlaidItem{}, pgx.ErrNoRows
	}
	if err != nil {
		return PlaidItem{}, fmt.Errorf("get plaid item: %w", err)
	}
	return item, nil
}

func (s *Store) UpdatePlaidCursor(ctx context.Context, plaidItemID int64, cursor string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE plaid_items SET transactions_cursor = $2, updated_at = now() WHERE id = $1`,
		plaidItemID, cursor,
	)
	if err != nil {
		return fmt.Errorf("update plaid cursor: %w", err)
	}
	return nil
}

func (s *Store) UpsertAccount(ctx context.Context, input AccountInput) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO bank_accounts (
			user_id, plaid_item_id, plaid_account_id, name, official_name, type, subtype, mask
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (plaid_account_id) DO UPDATE SET
			name = EXCLUDED.name,
			official_name = EXCLUDED.official_name,
			type = EXCLUDED.type,
			subtype = EXCLUDED.subtype,
			mask = EXCLUDED.mask,
			updated_at = now()
		RETURNING id`,
		input.UserID, input.PlaidItemID, input.PlaidAccountID, input.Name, input.OfficialName, input.Type, input.Subtype, input.Mask,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("upsert account: %w", err)
	}
	return id, nil
}

func (s *Store) GetAccountIDByPlaidAccountID(ctx context.Context, plaidAccountID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM bank_accounts WHERE plaid_account_id = $1`,
		plaidAccountID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, pgx.ErrNoRows
	}
	if err != nil {
		return 0, fmt.Errorf("get account by plaid account id: %w", err)
	}
	return id, nil
}

func (s *Store) UpsertTransaction(ctx context.Context, input TransactionInput) (StoredTransaction, error) {
	var tx StoredTransaction
	err := s.pool.QueryRow(ctx,
		`INSERT INTO transactions (
			user_id, bank_account_id, plaid_transaction_id, amount, iso_currency_code,
			transaction_date, pending, description_ciphertext, description_nonce,
			description_key_id, description_algorithm, merchant_name, source_environment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (plaid_transaction_id) DO UPDATE SET
			amount = EXCLUDED.amount,
			iso_currency_code = EXCLUDED.iso_currency_code,
			transaction_date = EXCLUDED.transaction_date,
			pending = EXCLUDED.pending,
			description_ciphertext = EXCLUDED.description_ciphertext,
			description_nonce = EXCLUDED.description_nonce,
			description_key_id = EXCLUDED.description_key_id,
			description_algorithm = EXCLUDED.description_algorithm,
			merchant_name = EXCLUDED.merchant_name,
			source_environment = EXCLUDED.source_environment,
			updated_at = now()
		RETURNING id, user_id, bank_account_id`,
		input.UserID, input.BankAccountID, input.PlaidTransactionID, input.Amount, input.ISOCurrencyCode,
		input.TransactionDate, input.Pending, input.DescriptionCiphertext, input.DescriptionNonce,
		input.DescriptionKeyID, input.DescriptionAlgorithm, input.MerchantName, input.SourceEnvironment,
	).Scan(&tx.ID, &tx.UserID, &tx.BankAccountID)
	if err != nil {
		return StoredTransaction{}, fmt.Errorf("upsert transaction: %w", err)
	}
	return tx, nil
}

func (s *Store) RecordWebhookEvent(ctx context.Context, input WebhookInput) (bool, error) {
	var inserted bool
	err := s.pool.QueryRow(ctx,
		`INSERT INTO webhook_events (event_key, item_id, webhook_type, webhook_code, payload)
		 VALUES ($1, $2, $3, $4, $5::jsonb)
		 ON CONFLICT (event_key) DO NOTHING
		 RETURNING true`,
		input.EventKey, input.ItemID, input.WebhookType, input.WebhookCode, string(input.Payload),
	).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("record webhook event: %w", err)
	}
	return inserted, nil
}

func (s *Store) MarkWebhookProcessed(ctx context.Context, eventKey string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE webhook_events SET processed_at = now() WHERE event_key = $1`,
		eventKey,
	)
	if err != nil {
		return fmt.Errorf("mark webhook processed: %w", err)
	}
	return nil
}

func (s *Store) GetTransactionByID(ctx context.Context, id int64) (Transaction, error) {
	var tx Transaction
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, bank_account_id, plaid_transaction_id, amount, iso_currency_code,
			transaction_date, pending, description_ciphertext, description_nonce,
			description_key_id, description_algorithm, merchant_name, source_environment,
			category, ml_confidence, categorized_at
		 FROM transactions WHERE id = $1`,
		id,
	).Scan(
		&tx.ID, &tx.UserID, &tx.BankAccountID, &tx.PlaidTransactionID, &tx.Amount, &tx.ISOCurrencyCode,
		&tx.TransactionDate, &tx.Pending, &tx.DescriptionCiphertext, &tx.DescriptionNonce,
		&tx.DescriptionKeyID, &tx.DescriptionAlgorithm, &tx.MerchantName, &tx.SourceEnvironment,
		&tx.Category, &tx.MLConfidence, &tx.CategorizedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, pgx.ErrNoRows
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("get transaction by id: %w", err)
	}
	return tx, nil
}

func (s *Store) UpdateTransactionCategory(ctx context.Context, id int64, category string, confidence float32) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE transactions
		    SET category = $2, ml_confidence = $3, categorized_at = now(), updated_at = now()
		  WHERE id = $1`,
		id, category, confidence,
	)
	if err != nil {
		return fmt.Errorf("update transaction category: %w", err)
	}
	return nil
}

func (s *Store) StoreArbitrageOpportunity(ctx context.Context, input ArbitrageInput) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO arbitrage_opportunities (
			user_id, transaction_id, merchant_name, category,
			current_amount, market_rate, savings_estimate, provider_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		input.UserID, input.TransactionID, input.MerchantName, input.Category,
		input.CurrentAmount, input.MarketRate, input.SavingsEstimate, input.ProviderURL,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store arbitrage opportunity: %w", err)
	}
	return id, nil
}
