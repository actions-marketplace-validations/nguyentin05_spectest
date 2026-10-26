package generate

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
)

// normalizeEvidence is the single canonical form used for every evidence
// check: line endings, NBSP and other whitespace collapse to one ASCII space,
// and both ends are trimmed.
func normalizeEvidence(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// containsEvidence reports whether needle (normalized) occurs in haystackNorm,
// which must already be normalized. An empty needle never matches.
func containsEvidence(haystackNorm, needle string) bool {
	n := normalizeEvidence(needle)
	return n != "" && strings.Contains(haystackNorm, n)
}

// newDelimiter returns a random delimiter that does not occur in any of data.
// It fences untrusted text (requirements) inside prompts.
func newDelimiter(tag string, data ...string) (string, error) {
	for range 8 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", fmt.Errorf("random delimiter: %w", err)
		}
		d := fmt.Sprintf("<<<%s_%x>>>", tag, b)
		clash := false
		for _, s := range data {
			if strings.Contains(s, d) {
				clash = true
				break
			}
		}
		if !clash {
			return d, nil
		}
	}
	return "", errors.New("cannot pick a unique delimiter")
}
