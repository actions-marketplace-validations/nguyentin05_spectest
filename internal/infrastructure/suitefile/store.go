package suitefile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nguyentin05/spectest/internal/domain"
	"github.com/nguyentin05/spectest/internal/platform/atomicfile"
)

// Fixed artifact names inside the output directory.
const (
	SuiteFileName     = "spectest.yaml"
	LogFileName       = "spectest.log.yaml"
	TrackFileName     = "spectest.track.yaml"
	PendingMarkerName = ".spectest.pending"
)

// Store loads and saves the three SpecTest state files.
type Store struct{}

// NewStore returns a Store.
func NewStore() *Store {
	return &Store{}
}

// pendingMarker is written before the artifacts and removed only after every
// requested write succeeds. It lets the next run detect an interrupted write.
type pendingMarker struct {
	Mode      string `json:"mode"`
	At        string `json:"at"`
	SuiteHash string `json:"suite_hash"`
	LogHash   string `json:"log_hash"`
	TrackHash string `json:"track_hash"`
}

// Load reads the three files in dir. Missing files yield nil members without
// creating or repairing anything. A pending marker aborts the load before any
// artifact is parsed.
func (s *Store) Load(ctx context.Context, dir string) (*domain.FileSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	markerPath := filepath.Join(dir, PendingMarkerName)
	if _, err := os.Stat(markerPath); err == nil {
		return nil, fmt.Errorf(
			"pending write marker %s exists: a previous write may have been interrupted; check %s, %s, and %s for consistency, then remove %s before retrying",
			markerPath, SuiteFileName, LogFileName, TrackFileName, markerPath,
		)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("check pending marker %s: %w", markerPath, err)
	}

	suite, err := loadArtifact(filepath.Join(dir, SuiteFileName), decodeSuite)
	if err != nil {
		return nil, err
	}
	log, err := loadArtifact(filepath.Join(dir, LogFileName), decodeLog)
	if err != nil {
		return nil, err
	}
	track, err := loadArtifact(filepath.Join(dir, TrackFileName), decodeTrack)
	if err != nil {
		return nil, err
	}

	return &domain.FileSet{Suite: suite, Log: log, Track: track}, nil
}

// Save encodes every requested file before touching the filesystem, then writes
// the pending marker, suite, log, and track in that order. The marker is
// removed only after all writes succeed; any failure leaves it in place.
func (s *Store) Save(ctx context.Context, dir string, files *domain.FileSet, opts domain.WriteOptions) error {
	if !opts.Suite && !opts.Log && !opts.Track {
		return nil
	}
	if files == nil {
		return errors.New("suitefile: cannot save: file set is nil")
	}

	type artifact struct {
		name string
		data []byte
	}

	artifacts := make([]artifact, 0, 3)
	if opts.Suite {
		if files.Suite == nil {
			return fmt.Errorf("suitefile: cannot save: %s is nil", SuiteFileName)
		}
		data, err := encodeSuite(files.Suite)
		if err != nil {
			return fmt.Errorf("%s: encode: %w", SuiteFileName, err)
		}
		artifacts = append(artifacts, artifact{name: SuiteFileName, data: data})
	}
	if opts.Log {
		if files.Log == nil {
			return fmt.Errorf("suitefile: cannot save: %s is nil", LogFileName)
		}
		data, err := encodeLog(files.Log)
		if err != nil {
			return fmt.Errorf("%s: encode: %w", LogFileName, err)
		}
		artifacts = append(artifacts, artifact{name: LogFileName, data: data})
	}
	if opts.Track {
		if files.Track == nil {
			return fmt.Errorf("suitefile: cannot save: %s is nil", TrackFileName)
		}
		data, err := encodeTrack(files.Track)
		if err != nil {
			return fmt.Errorf("%s: encode: %w", TrackFileName, err)
		}
		artifacts = append(artifacts, artifact{name: TrackFileName, data: data})
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", dir, err)
	}

	marker := pendingMarker{Mode: opts.Mode, At: time.Now().UTC().Format(time.RFC3339)}
	for _, a := range artifacts {
		switch a.name {
		case SuiteFileName:
			marker.SuiteHash = hash12(a.data)
		case LogFileName:
			marker.LogHash = hash12(a.data)
		case TrackFileName:
			marker.TrackHash = hash12(a.data)
		}
	}
	markerData, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode pending marker: %w", err)
	}
	markerData = append(markerData, '\n')

	markerPath := filepath.Join(dir, PendingMarkerName)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := atomicfile.Write(markerPath, markerData, 0o644); err != nil {
		return fmt.Errorf("write pending marker %s: %w", markerPath, err)
	}

	for _, a := range artifacts {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%s: %w (pending marker retained at %s)", a.name, err, markerPath)
		}
		if err := atomicfile.Write(filepath.Join(dir, a.name), a.data, 0o644); err != nil {
			return fmt.Errorf("%s: write: %w (pending marker retained at %s; check all three files before removing it)", a.name, err, markerPath)
		}
	}

	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("remove pending marker %s: %w", markerPath, err)
	}
	return nil
}

// loadArtifact reads and decodes one artifact. A missing file is not an error
// and yields a nil result.
func loadArtifact[T any](path string, decode func([]byte) (*T, error)) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	value, err := decode(data)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	return value, nil
}

// hash12 returns the first 12 hexadecimal characters of the SHA-256 digest.
func hash12(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}
