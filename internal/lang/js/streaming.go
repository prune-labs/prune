package js

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"
	"prune/internal/config"
	"prune/internal/rules"
	"prune/internal/scan"
)

type StreamHandler func([]rules.Finding) error

func AnalyzeStreaming(ctx context.Context, cfg *config.Config, handler StreamHandler) ([]rules.Finding, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	stream := cfg.Scan.Stream
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	entries, err := scan.CollectWithContext(ctx, cfg)
	if err != nil {
		return nil, err
	}

	entriesCh := make(chan []scan.FileEntry, 1)
	resultsCh := make(chan []rules.Finding, 1)

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		defer close(entriesCh)
		batch := []scan.FileEntry{}
		lastEmit := time.Now()
		interval := time.Duration(stream.IntervalMs) * time.Millisecond

		for _, entry := range entries {
			select {
			case <-groupCtx.Done():
				return groupCtx.Err()
			default:
			}

			batch = append(batch, entry)

			batchSize := 50
			if stream.BatchSize > 0 {
				batchSize = stream.BatchSize
			}
			if len(batch) >= batchSize || time.Since(lastEmit) >= interval {
				select {
				case <-groupCtx.Done():
					return groupCtx.Err()
				case entriesCh <- batch:
				}
				batch = []scan.FileEntry{}
				lastEmit = time.Now()
			}
		}

		if len(batch) > 0 {
			select {
			case <-groupCtx.Done():
				return groupCtx.Err()
			case entriesCh <- batch:
			}
		}

		return nil
	})

	group.Go(func() error {
		collector := NewCollector(cfg)
		var lastCollected *Collected
		emitted := map[string]bool{}

		for {
			select {
			case <-groupCtx.Done():
				return groupCtx.Err()
			case batch, ok := <-entriesCh:
				if !ok {
					if lastCollected == nil {
						resultsCh <- nil
						return nil
					}
					resultsCh <- applyRules(cfg, lastCollected)
					return nil
				}
				collected, err := collector.Collect(groupCtx, batch)
				if err != nil {
					return err
				}
				lastCollected = collected
				findings := applyRules(cfg, collected)

				if stream.Enabled && stream.IntervalMs > 0 && handler != nil {
					newFindings := make([]rules.Finding, 0, len(findings))
					for _, finding := range findings {
						if emitted[finding.ID] {
							continue
						}
						emitted[finding.ID] = true
						newFindings = append(newFindings, finding)
					}
					if len(newFindings) == 0 {
						continue
					}
					if err := handler(newFindings); err != nil {
						return err
					}
				}
			}
		}
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	result := <-resultsCh
	return result, nil
}
