package generate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

// serverNamePattern is the accepted `--api` server identifier.
var serverNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// parseAPIFlags converts repeated `--api <server>=<path>` values into ordered
// APIInput values. The first `=` separates name from path, so paths may contain
// further equals signs. Input order is preserved because it defines `servers` order.
func parseAPIFlags(values []string) ([]domain.APIInput, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one --api <server>=<path> is required")
	}

	inputs := make([]domain.APIInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		server, path, ok := strings.Cut(value, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --api value %q: expected <server>=<path>", value)
		}
		server = strings.TrimSpace(server)
		if !serverNamePattern.MatchString(server) {
			return nil, fmt.Errorf("invalid --api value %q: server name must match ^[a-z][a-z0-9_-]*$", value)
		}
		if path == "" {
			return nil, fmt.Errorf("invalid --api value %q: path must not be empty", value)
		}
		if _, exists := seen[server]; exists {
			return nil, fmt.Errorf("duplicate --api server %q", server)
		}
		seen[server] = struct{}{}
		inputs = append(inputs, domain.APIInput{Server: server, Path: path})
	}

	return inputs, nil
}
