package generate

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nguyentin05/spectest/internal/infrastructure/suitefile"
)

// outputState reports the existing artifacts in dir and, if present, the path
// of the pending write marker left behind by an interrupted write.
func outputState(dir string) (existing []string, pending string, err error) {
	markerPath := filepath.Join(dir, suitefile.PendingMarkerName)
	if _, statErr := os.Stat(markerPath); statErr == nil {
		pending = markerPath
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("check pending marker %s: %w", markerPath, statErr)
	}

	for _, name := range []string{suitefile.SuiteFileName, suitefile.LogFileName, suitefile.TrackFileName} {
		path := filepath.Join(dir, name)
		if _, statErr := os.Stat(path); statErr == nil {
			existing = append(existing, path)
		} else if !errors.Is(statErr, fs.ErrNotExist) {
			return nil, "", fmt.Errorf("check output file %s: %w", path, statErr)
		}
	}

	return existing, pending, nil
}

// confirmOverwrite asks whether the existing artifacts may be overwritten.
// Only "y" or "yes" (case-insensitively) confirm; empty input or EOF declines.
func confirmOverwrite(in io.Reader, out io.Writer, paths []string) (bool, error) {
	if _, err := fmt.Fprintln(out, "Existing output files:"); err != nil {
		return false, err
	}
	for _, path := range paths {
		if _, err := fmt.Fprintf(out, "  %s\n", path); err != nil {
			return false, err
		}
	}
	if _, err := fmt.Fprint(out, "Overwrite these files? [y/N] "); err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read confirmation: %w", err)
		}
		return false, nil
	}

	switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// guardOverwrite aborts when an interrupted write left a pending marker and
// asks before overwriting existing artifacts. Dry runs never prompt and never
// write, but a pending marker still aborts them.
func guardOverwrite(in io.Reader, out io.Writer, dir string, dryRun bool) error {
	existing, pending, err := outputState(dir)
	if err != nil {
		return err
	}
	if pending != "" {
		return fmt.Errorf(
			"pending write marker %s exists; check the three files and remove the marker before retrying",
			pending,
		)
	}
	if dryRun || len(existing) == 0 {
		return nil
	}

	ok, err := confirmOverwrite(in, out, existing)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("aborted: existing output files were not overwritten")
	}
	return nil
}
