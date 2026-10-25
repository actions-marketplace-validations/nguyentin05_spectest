package generate

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
)

// Mode identifies the outcome of one Execute call.
type Mode string

const (
	ModeInitial      Mode = "initial"
	ModeUpdate       Mode = "update"
	ModeNoop         Mode = "no-op"
	ModeSync         Mode = "sync"
	ModeRecoverTrack Mode = "recover-track"
	ModeDryRun       Mode = "dry-run"
)

// buildBaseEntry constructs the single base entry for spectest.log.yaml.
// The suite snapshot is deep-cloned so later mutations cannot alter the log.
func (g *Generator) buildBaseEntry(s *domain.Suite) domain.BaseEntry {
	return domain.BaseEntry{
		At:        g.clock.Now().Format(time.RFC3339),
		By:        "ai",
		Spectest:  cloneSuite(s),
		AfterHash: suiteHash(s),
	}
}

// buildTrack constructs spectest.track.yaml.
//
// Initial run (old == nil): every current API is "active", last_update = 0.
// Update run (old != nil): removed entries from old are preserved forever;
// current APIs are refreshed; source fingerprints are reset.
func buildTrack(src *domain.SourceSet, old *domain.TrackFile, lastUpdate int) *domain.TrackFile {
	apis := make(map[string]domain.APIState, len(src.APIs))

	// Preserve old states (including removed) so history is not lost.
	if old != nil {
		for k, v := range old.APIs {
			apis[k] = v
		}
	}

	// Refresh entries for APIs currently present in source.
	present := make(map[string]struct{}, len(src.APIs))
	for _, a := range src.APIs {
		present[a.Key] = struct{}{}
		apis[a.Key] = domain.APIState{
			Status: "active",
			Hash:   a.Hash,
			Sig:    a.Sig,
		}
	}

	// Mark previously-active entries that are no longer present as removed.
	for k, v := range apis {
		if _, ok := present[k]; !ok && v.Status == "active" {
			v.Status = "removed"
			apis[k] = v
		}
	}

	sources := domain.Sources{
		OpenAPI: src.OpenAPI,
	}
	if src.Requirements != nil {
		sources.Requirements = &domain.RequirementState{
			File: src.Requirements.File,
			Raw:  src.Requirements.Raw,
			Hash: requirementsHash(src.Requirements.Raw),
		}
	}

	return &domain.TrackFile{
		LastUpdate: lastUpdate,
		Sources:    sources,
		APIs:       apis,
	}
}

// persist delegates to store.Save. On dry-run, it returns immediately without
// touching the filesystem (no directory, no marker, no file).
func (g *Generator) persist(
	ctx context.Context, req Request, files *domain.FileSet, opts domain.WriteOptions,
) error {
	if req.DryRun {
		return nil
	}
	return g.store.Save(ctx, req.OutputDir, files, opts)
}

// joinDisplayPath returns the path as it should appear in the CLI summary.
// Relative paths get a "./" prefix for clarity, absolute paths are kept as-is.
func joinDisplayPath(dir, name string) string {
	joined := filepath.Join(dir, name)
	if filepath.IsAbs(joined) {
		return joined
	}
	if !strings.HasPrefix(joined, "./") {
		return "./" + joined
	}
	return joined
}

// buildInitialFileSet assembles the three files for a first run.
func (g *Generator) buildInitialFileSet(src *domain.SourceSet, suite *domain.Suite) (*domain.FileSet, error) {
	normalized := normalizeSuite(suite)
	base := g.buildBaseEntry(normalized)
	return &domain.FileSet{
		Suite: normalized,
		Log: &domain.LogFile{
			Base:   base,
			Update: []domain.UpdateEntry{},
		},
		Track: buildTrack(src, nil, 0),
	}, nil
}
