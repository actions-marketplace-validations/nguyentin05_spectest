// Package atomicfile writes files atomically via a temporary file and rename.
package atomicfile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Write atomically writes data to path with the given permission bits. It
// creates a temporary file in the same directory as path (never os.TempDir),
// applies perm, writes the full payload, syncs, closes, and renames the
// temporary file over path. On any error it removes the temporary file before
// returning, so readers either see the old content or the complete new
// content, never a partial write.
func Write(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("atomicfile: create temp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	if err := writeSyncClose(tmp, data, perm); err != nil {
		tmp.Close() // best-effort; the error is already set
		os.Remove(tmpName)
		return fmt.Errorf("atomicfile: write %s: %w", path, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomicfile: rename to %s: %w", path, err)
	}
	return nil
}

func writeSyncClose(tmp *os.File, data []byte, perm fs.FileMode) error {
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}
