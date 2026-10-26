package generate

import (
	"fmt"
	"io"
	"strings"

	usecasegenerate "github.com/nguyentin05/spectest/internal/usecase/generate"
)

// printSummary writes the outcome of a successful generate run: the mode and
// input source used, the servers and API count discovered, any pipeline
// warnings, the three output paths, and the dry-run marker when no files were
// written.
func printSummary(w io.Writer, res *usecasegenerate.Result) {
	fmt.Fprintf(w, "Mode: %s\n", res.Mode)
	fmt.Fprintf(w, "Input: %s\n", res.Input)
	fmt.Fprintf(w, "Servers: %s\n", strings.Join(res.Servers, ", "))
	fmt.Fprintf(w, "APIs: %d\n", res.APITotal)
	for _, warning := range res.Warnings {
		fmt.Fprintf(w, "Warning: %s\n", warning)
	}
	fmt.Fprintf(w, "Suite: %s\n", res.SuitePath)
	fmt.Fprintf(w, "Log: %s\n", res.LogPath)
	fmt.Fprintf(w, "Track: %s\n", res.TrackPath)
	if res.NoFilesWritten {
		fmt.Fprintln(w, "no files written (dry-run)")
	}
}
