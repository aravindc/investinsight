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
}

// BackfillAll fetches and stores a logo for every instrument that doesn't
// have one yet. Instruments logo.dev has no logo for (ErrNotFound) are
// counted as failed, same as any other fetch error — the frontend's
// initials-badge fallback covers those, and retrying them daily is
// harmless (a coverage gap on logo.dev's end may later be filled).
func (s *Service) BackfillAll(ctx context.Context) (*BackfillResult, error) {
	instruments, err := s.repo.InstrumentsMissingLogo(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instruments missing a logo: %w", err)
	}

	result := &BackfillResult{}
	for _, inst := range instruments {
		symbol := marketprice.YahooSymbol(inst.Ticker, inst.BaseCurrency, inst.ISIN)

		image, err := s.client.FetchLogo(ctx, symbol)
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
