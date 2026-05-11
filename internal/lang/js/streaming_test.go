package js

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"prune/internal/config"
	"prune/internal/rules"
)

func TestAnalyzeStreamingMatchesNonStreaming(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cfg := &config.Config{Version: 1}
	cfg.Scan.Paths = []string{filepath.Join(root, "internal", "cli", "testdata", "streaming", "src")}
	cfg.Scan.Include = []string{"**/*.ts"}
	cfg.Scan.Stream.Enabled = true
	cfg.Scan.Stream.IntervalMs = 1
	cfg.Scan.Stream.BatchSize = 1

	var streamedBatches [][]rules.Finding
	cfg.Entrypoints.Files = []string{
		filepath.ToSlash(cfg.Scan.Paths[0]) + "/index.ts",
	}

	_, err = AnalyzeStreaming(context.Background(), cfg, func(batch []rules.Finding) error {
		streamedBatches = append(streamedBatches, batch)
		return nil
	})
	if err != nil {
		t.Fatalf("streaming analyze: %v", err)
	}

	streamedFinal, err := AnalyzeStreaming(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("streaming final analyze: %v", err)
	}

	seen := map[string]int{}
	for _, batch := range streamedBatches {
		for _, finding := range batch {
			seen[finding.ID]++
		}
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("expected finding %q once in streamed batches, got %d", id, count)
		}
	}

	finalSeen := map[string]int{}
	for _, finding := range streamedFinal {
		finalSeen[finding.ID]++
	}
	for id, count := range finalSeen {
		if count != 1 {
			t.Fatalf("expected final findings to be unique for %q, got %d", id, count)
		}
	}
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "..")), nil
}
