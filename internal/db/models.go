package db

import "time"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type PlaidItem struct {
	ID                    int64
	UserID                int64
	ItemID                string
	AccessTokenCiphertext string
	AccessTokenNonce      string
	AccessTokenKeyID      string
	AccessTokenAlgorithm  string
	TransactionsCursor    string
}

type AccountInput struct {
	UserID         int64
	PlaidItemID    int64
	PlaidAccountID string
	Name           string
	OfficialName   string
	Type           string
	Subtype        string
	Mask           string
}

type TransactionInput struct {
	UserID                 int64
	BankAccountID         int64
	PlaidTransactionID     string
	Amount                 float64
	ISOCurrencyCode        string
	TransactionDate        time.Time
	Pending                bool
	DescriptionCiphertext  string
	DescriptionNonce       string
	DescriptionKeyID       string
	DescriptionAlgorithm   string
	MerchantName           string
	SourceEnvironment     string
}

type StoredTransaction struct {
	ID            int64
	UserID        int64
	BankAccountID int64
}

type Transaction struct {
	ID                    int64
	UserID                int64
	BankAccountID         int64
	PlaidTransactionID    string
	Amount                float64
	ISOCurrencyCode       string
	TransactionDate       time.Time
	Pending               bool
	DescriptionCiphertext string
	DescriptionNonce      string
	DescriptionKeyID      string
	DescriptionAlgorithm  string
	MerchantName          string
	SourceEnvironment     string
	Category              *string
	MLConfidence          *float32
	CategorizedAt         *time.Time
}

type WebhookInput struct {
	EventKey    string
	ItemID      string
	WebhookType string
	WebhookCode string
	Payload     []byte
}

type ArbitrageInput struct {
	UserID          int64
	TransactionID   int64
	MerchantName    string
	Category        string
	CurrentAmount   float64
	MarketRate      *float64
	SavingsEstimate *float64
	ProviderURL     string
}

