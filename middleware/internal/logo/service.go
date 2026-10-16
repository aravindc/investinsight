package logo

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"investinsight/middleware/internal/marketprice"
)

type Service struct {
	repo   *Repository
	client *Client
	logger *zap.Logger
}

func NewService(repo *Repository, client *Client, logger *zap.Logger) *Service {
	return &Service{repo: repo, client: client, logger: logger}
}

// BackfillResult summarizes the outcome of a backfill run.
type BackfillResult struct {
	Processed int
	Failed    []string // tickers that could not be fetched/stored
	// Removed counts stored logos dropped on a refresh because logo.dev no
	// longer has one (e.g. placeholders saved before fallback=404).
	Removed int
}

// BackfillAll fetches and stores a logo for every instrument that doesn't
// have one yet. Instruments logo.dev has no logo for (ErrNotFound) are
// counted as failed, same as any other fetch error — the frontend's
// initials-badge fallback covers those, and retrying them daily is
// harmless (a coverage gap on logo.dev's end may later be filled).
//
// With refresh, every instrument is fetched again: changed logos are
// replaced, and ones logo.dev has no logo for are deleted.
func (s *Service) BackfillAll(ctx context.Context, refresh bool) (*BackfillResult, error) {
	list := s.repo.InstrumentsMissingLogo
	if refresh {
		list = s.repo.AllInstruments
	}
	instruments, err := list(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instruments to fetch logos for: %w", err)
	}

	result := &BackfillResult{}
	for _, inst := range instruments {
		symbol := marketprice.YahooSymbol(inst.Ticker, inst.BaseCurrency, inst.ISIN)

		image, _, err := s.client.FetchLogoWithLSEFallback(ctx, symbol)
		if refresh && errors.Is(err, ErrNotFound) {
			removed, delErr := s.repo.DeleteLogo(ctx, inst.ID)
			if delErr != nil {
				s.logger.Error("failed to remove logo", zap.String("ticker", inst.Ticker), zap.Error(delErr))
			} else if removed {
				result.Removed++
			}
		}
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				s.logger.Warn("failed to fetch logo", zap.String("ticker", inst.Ticker), zap.String("symbol", symbol), zap.Error(err))
			}
			result.Failed = append(result.Failed, inst.Ticker)
			continue
		}

		if err := s.repo.UpsertLogo(ctx, inst.ID, image.Data, image.ContentType); err != nil {
			s.logger.Error("failed to store logo", zap.String("ticker", inst.Ticker), zap.Error(err))
			result.Failed = append(result.Failed, inst.Ticker)
			continue
		}
		result.Processed++
	}

	return result, nil
}
