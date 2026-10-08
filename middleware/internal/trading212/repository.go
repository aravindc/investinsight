package trading212

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	SyncIdle      = "idle"
	SyncRunning   = "running"
	SyncSucceeded = "succeeded"
	SyncFailed    = "failed"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Connection is an account's Trading 212 link as exposed to the API — the
// encrypted credentials are deliberately not part of it.
type Connection struct {
	AccountID      uuid.UUID  `json:"accountId"`
	KeyHint        string     `json:"keyHint"`
	T212AccountID  *int64     `json:"t212AccountId"`
	Currency       *string    `json:"currency"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	SyncStatus     string     `json:"syncStatus"`
	SyncMessage    *string    `json:"syncMessage"`
	SyncStartedAt  *time.Time `json:"syncStartedAt"`
	SyncFinishedAt *time.Time `json:"syncFinishedAt"`
	SyncedThrough  *time.Time `json:"syncedThrough"`
	LastInserted   int        `json:"lastInserted"`
	LastDuplicate  int        `json:"lastDuplicate"`
}

const connectionColumns = `
	account_id, key_hint, t212_account_id, currency, updated_at,
	sync_status, sync_message, sync_started_at, sync_finished_at,
	synced_through, last_inserted, last_duplicate`

func scanConnection(row interface{ Scan(...any) error }) (*Connection, error) {
	var c Connection
	err := row.Scan(&c.AccountID, &c.KeyHint, &c.T212AccountID, &c.Currency, &c.UpdatedAt,
		&c.SyncStatus, &c.SyncMessage, &c.SyncStartedAt, &c.SyncFinishedAt,
		&c.SyncedThrough, &c.LastInserted, &c.LastDuplicate)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) List(ctx context.Context) ([]Connection, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+connectionColumns+` FROM trading212_connections;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// Get returns the account's connection, or nil if it has none.
func (r *Repository) Get(ctx context.Context, accountID uuid.UUID) (*Connection, error) {
	c, err := scanConnection(r.db.QueryRowContext(ctx,
		`SELECT `+connectionColumns+` FROM trading212_connections WHERE account_id = $1;`, accountID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// Upsert stores (or replaces) the account's encrypted credentials. Replacing
// keeps the sync history, so a key rotation doesn't trigger a full re-sync.
func (r *Repository) Upsert(ctx context.Context, accountID uuid.UUID, keyEnc, secretEnc []byte, keyHint string, t212AccountID int64, currency string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trading212_connections
			(account_id, api_key_encrypted, api_secret_encrypted, key_hint, t212_account_id, currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (account_id) DO UPDATE SET
			api_key_encrypted = EXCLUDED.api_key_encrypted,
			api_secret_encrypted = EXCLUDED.api_secret_encrypted,
			key_hint = EXCLUDED.key_hint,
			t212_account_id = EXCLUDED.t212_account_id,
			currency = EXCLUDED.currency,
			updated_at = NOW();`,
		accountID, keyEnc, secretEnc, keyHint, t212AccountID, currency)
	return err
}

func (r *Repository) Delete(ctx context.Context, accountID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trading212_connections WHERE account_id = $1;`, accountID)
	return err
}

// EncryptedCredentials returns the stored ciphertexts for accountID.
func (r *Repository) EncryptedCredentials(ctx context.Context, accountID uuid.UUID) (keyEnc, secretEnc []byte, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT api_key_encrypted, api_secret_encrypted
		FROM trading212_connections WHERE account_id = $1;`, accountID,
	).Scan(&keyEnc, &secretEnc)
	return keyEnc, secretEnc, err
}

// MarkSyncStarted atomically flips the connection to "running", returning
// false if it was already running (so two syncs can't overlap).
func (r *Repository) MarkSyncStarted(ctx context.Context, accountID uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE trading212_connections
		SET sync_status = $2, sync_message = NULL, sync_started_at = NOW(),
			sync_finished_at = NULL, last_inserted = 0, last_duplicate = 0
		WHERE account_id = $1 AND sync_status <> $2;`, accountID, SyncRunning)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// RecordProgress stores a progress message, the running totals and how far
// the account is now synced.
func (r *Repository) RecordProgress(ctx context.Context, accountID uuid.UUID, message string, syncedThrough time.Time, inserted, duplicate int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trading212_connections
		SET sync_message = $2, synced_through = $3, last_inserted = $4, last_duplicate = $5
		WHERE account_id = $1;`, accountID, message, syncedThrough, inserted, duplicate)
	return err
}

func (r *Repository) SetSyncMessage(ctx context.Context, accountID uuid.UUID, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trading212_connections SET sync_message = $2 WHERE account_id = $1;`, accountID, message)
	return err
}

func (r *Repository) MarkSyncFinished(ctx context.Context, accountID uuid.UUID, status, message string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trading212_connections
		SET sync_status = $2, sync_message = $3, sync_finished_at = NOW()
		WHERE account_id = $1;`, accountID, status, message)
	return err
}

// FailInterruptedSyncs marks syncs left "running" by a previous process
// (e.g. a restart mid-sync) as failed, so they can be re-run.
func (r *Repository) FailInterruptedSyncs(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trading212_connections
		SET sync_status = $1, sync_message = 'Interrupted by a server restart; run sync again.', sync_finished_at = NOW()
		WHERE sync_status = $2;`, SyncFailed, SyncRunning)
	return err
}

// scopedAccount is an account in a CashScope and whether it has a Trading
// 212 connection.
type scopedAccount struct {
	ID        uuid.UUID
	Name      string
	Connected bool
}

// AccountsInScope lists the accounts a CashScope covers, ordered by name.
func (r *Repository) AccountsInScope(ctx context.Context, scope CashScope) ([]scopedAccount, error) {
	where, arg := "a.id = $1", scope.AccountID
	if scope.Household {
		where, arg = "a.household_id = $1", scope.HouseholdID
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.name, c.account_id IS NOT NULL
		FROM accounts a
		LEFT JOIN trading212_connections c ON c.account_id = a.id
		WHERE `+where+`
		ORDER BY a.name;`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []scopedAccount
	for rows.Next() {
		var a scopedAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.Connected); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// LatestTransactionTime returns the newest imported transaction for the
// account (e.g. from earlier CSV uploads), or a zero time if there are none.
func (r *Repository) LatestTransactionTime(ctx context.Context, accountID uuid.UUID) (time.Time, error) {
	var t sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT MAX(transaction_time) FROM source_transactions WHERE account_id = $1;`, accountID,
	).Scan(&t)
	return t.Time, err
}
