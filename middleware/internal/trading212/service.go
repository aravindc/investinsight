package trading212

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"investinsight/middleware/internal/ingest"
)

// ErrSyncRunning is returned when a sync is already in progress for the account.
var ErrSyncRunning = errors.New("a sync is already running for this account")

// ErrNotConnected is returned when the account has no Trading 212 credentials.
var ErrNotConnected = errors.New("account is not connected to Trading 212")

// ErrUnsupportedCurrency is returned for Trading 212 accounts whose primary
// currency isn't GBP: the ledger records every amount in GBP.
var ErrUnsupportedCurrency = errors.New("only GBP Trading 212 accounts are supported")

const (
	// exportWindow bounds each export request; Trading 212's history export
	// is limited to roughly a year per report.
	exportWindow = 365 * 24 * time.Hour
	// syncTimeout caps a whole sync (several windows, each waiting on report
	// generation under tight rate limits).
	syncTimeout = 3 * time.Hour
	// reportTimeout caps how long one report may take to generate.
	reportTimeout = 30 * time.Minute
)

// defaultSyncStart is where a first sync begins when the account has no
// transactions yet. Earlier history can still be added by CSV upload.
var defaultSyncStart = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

type Service struct {
	repo     *Repository
	client   *Client
	cipher   *Cipher
	engine   *ingest.Engine
	holdings HoldingsSummarizer
	logger   *zap.Logger

	summaries summaryCache

	// Minimum gaps between calls, from Trading 212's rate limits. Fields
	// (not constants) so tests can shrink them.
	requestExportInterval time.Duration
	listExportsInterval   time.Duration
}

func NewService(repo *Repository, client *Client, cipher *Cipher, engine *ingest.Engine, holdings HoldingsSummarizer, logger *zap.Logger) *Service {
	return &Service{
		repo:                  repo,
		client:                client,
		cipher:                cipher,
		engine:                engine,
		holdings:              holdings,
		logger:                logger,
		requestExportInterval: 31 * time.Second,
		listExportsInterval:   61 * time.Second,
	}
}

// Connect validates the credentials against Trading 212 and stores them
// encrypted for accountID.
func (s *Service) Connect(ctx context.Context, accountID uuid.UUID, creds Credentials) (*Connection, error) {
	if s.cipher == nil {
		return nil, ErrNoEncryptionKey
	}

	summary, err := s.client.AccountSummary(ctx, creds)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(summary.Currency, "GBP") {
		return nil, fmt.Errorf("%w (this account is in %s)", ErrUnsupportedCurrency, summary.Currency)
	}

	aad := []byte(accountID.String())
	keyEnc, err := s.cipher.Encrypt([]byte(creds.APIKey), aad)
	if err != nil {
		return nil, err
	}
	secretEnc, err := s.cipher.Encrypt([]byte(creds.APISecret), aad)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Upsert(ctx, accountID, keyEnc, secretEnc, keyHint(creds.APIKey), summary.ID, strings.ToUpper(summary.Currency)); err != nil {
		return nil, err
	}
	s.summaries.set(accountID, summary) // a new key may point at a different T212 account
	return s.repo.Get(ctx, accountID)
}

func keyHint(apiKey string) string {
	if len(apiKey) <= 4 {
		return apiKey
	}
	return apiKey[len(apiKey)-4:]
}

// StartSync begins a background sync for accountID and returns immediately.
// Progress and the outcome are recorded on the connection row.
func (s *Service) StartSync(ctx context.Context, accountID uuid.UUID) error {
	conn, err := s.repo.Get(ctx, accountID)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrNotConnected
	}
	creds, err := s.credentials(ctx, accountID)
	if err != nil {
		return err
	}

	started, err := s.repo.MarkSyncStarted(ctx, accountID)
	if err != nil {
		return err
	}
	if !started {
		return ErrSyncRunning
	}

	from, err := s.syncStart(ctx, accountID)
	if err != nil {
		_ = s.repo.MarkSyncFinished(ctx, accountID, SyncFailed, err.Error())
		return err
	}

	go s.runSync(accountID, creds, from)
	return nil
}

func (s *Service) credentials(ctx context.Context, accountID uuid.UUID) (Credentials, error) {
	keyEnc, secretEnc, err := s.repo.EncryptedCredentials(ctx, accountID)
	if err != nil {
		return Credentials{}, err
	}
	aad := []byte(accountID.String())
	key, err := s.cipher.Decrypt(keyEnc, aad)
	if err != nil {
		return Credentials{}, err
	}
	secret, err := s.cipher.Decrypt(secretEnc, aad)
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{APIKey: string(key), APISecret: string(secret)}, nil
}

// syncStart is the account's newest transaction already in the database
// (from earlier syncs or CSV uploads), or defaultSyncStart if it has none.
// This assumes the history before that point is complete: anything Trading
// 212 later back-dates to before it won't be fetched. The newest
// transaction itself is re-exported and skipped as a duplicate on ingest.
func (s *Service) syncStart(ctx context.Context, accountID uuid.UUID) (time.Time, error) {
	latest, err := s.repo.LatestTransactionTime(ctx, accountID)
	if err != nil {
		return time.Time{}, err
	}
	if latest.IsZero() {
		return defaultSyncStart, nil
	}
	return latest, nil
}

type window struct{ from, to time.Time }

// exportWindows splits [from, to) into consecutive windows of at most size.
func exportWindows(from, to time.Time, size time.Duration) []window {
	var out []window
	for start := from; start.Before(to); start = start.Add(size) {
		end := start.Add(size)
		if end.After(to) {
			end = to
		}
		out = append(out, window{start, end})
	}
	return out
}

func (s *Service) runSync(accountID uuid.UUID, creds Credentials, from time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	log := s.logger.With(zap.String("accountId", accountID.String()))
	// Whatever was imported (even by a sync that fails partway) may have
	// moved cash, so drop the cached summary when the sync ends.
	defer s.summaries.forget(accountID)
	windows := exportWindows(from, time.Now().UTC(), exportWindow)
	log.Info("trading 212 sync started", zap.Time("from", from), zap.Int("windows", len(windows)))

	limiter := &rateLimiter{}
	inserted, duplicate := 0, 0
	for i, w := range windows {
		progress := fmt.Sprintf("Exporting %s to %s (%d of %d)…",
			w.from.Format("2 Jan 2006"), w.to.Format("2 Jan 2006"), i+1, len(windows))
		_ = s.repo.SetSyncMessage(ctx, accountID, progress)

		csv, err := s.exportWindow(ctx, limiter, creds, w)
		if err == nil {
			var result *ingest.Result
			result, err = s.engine.LoadCSV(ctx, bytes.NewReader(csv), true, accountID)
			if err == nil {
				inserted += result.Inserted
				duplicate += result.Duplicate
				// Rejected rows would leave a gap the next sync (which starts
				// from the newest stored transaction) can't fill, so fail.
				if n := len(result.Errors); n > 0 {
					err = fmt.Errorf("%d row(s) in the export couldn't be imported; first: %v", n, result.Errors[0])
				}
			}
		}
		if err != nil {
			log.Error("trading 212 sync failed", zap.Error(err))
			_ = s.repo.MarkSyncFinished(context.Background(), accountID, SyncFailed, err.Error())
			return
		}

		if err := s.repo.RecordProgress(ctx, accountID, progress, w.to, inserted, duplicate); err != nil {
			log.Error("recording trading 212 sync progress", zap.Error(err))
		}
	}

	msg := fmt.Sprintf("%d new transaction(s) imported, %d duplicate(s) skipped.", inserted, duplicate)
	_ = s.repo.MarkSyncFinished(context.Background(), accountID, SyncSucceeded, msg)
	log.Info("trading 212 sync finished", zap.Int("inserted", inserted), zap.Int("duplicate", duplicate))
}

// exportWindow requests a report for w, waits for it to finish and returns
// the downloaded CSV.
func (s *Service) exportWindow(ctx context.Context, limiter *rateLimiter, creds Credentials, w window) ([]byte, error) {
	var reportID int64
	err := limiter.call(ctx, "request", s.requestExportInterval, func() (err error) {
		reportID, err = s.client.RequestExport(ctx, creds, w.from, w.to)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("requesting export: %w", err)
	}

	deadline := time.Now().Add(reportTimeout)
	for time.Now().Before(deadline) {
		var reports []Report
		err := limiter.call(ctx, "list", s.listExportsInterval, func() (err error) {
			reports, err = s.client.ListExports(ctx, creds)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("checking export status: %w", err)
		}

		for _, r := range reports {
			if r.ReportID != reportID {
				continue
			}
			switch r.Status {
			case ReportFinished:
				if r.DownloadLink == "" {
					return nil, errors.New("export finished without a download link")
				}
				return s.client.Download(ctx, r.DownloadLink)
			case ReportFailed, ReportCanceled:
				return nil, fmt.Errorf("trading 212 export %d ended with status %s", reportID, r.Status)
			}
		}
	}
	return nil, fmt.Errorf("trading 212 export %d did not finish within %s", reportID, reportTimeout)
}

// rateLimiter spaces calls to each Trading 212 endpoint by at least its
// minimum interval, and retries once after a full interval on a 429.
type rateLimiter struct {
	last map[string]time.Time
}

func (l *rateLimiter) call(ctx context.Context, key string, interval time.Duration, fn func() error) error {
	if l.last == nil {
		l.last = make(map[string]time.Time)
	}
	for attempt := 0; ; attempt++ {
		if wait := time.Until(l.last[key].Add(interval)); wait > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}
		l.last[key] = time.Now()
		err := fn()
		if errors.Is(err, ErrRateLimited) && attempt < 3 {
			continue // l.last was just reset, so the next attempt waits a full interval
		}
		return err
	}
}
