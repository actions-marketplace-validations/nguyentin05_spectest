package generate

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

// resolveSources validates the request, loads the sources and fixes the input
// mode. Everything here happens before any LLM call.
//
//	--api only        -> openapi
//	--api + --req     -> both
//
// A run without --api is rejected: requirements never define APIs.
func (g *Generator) resolveSources(ctx context.Context, req Request) (*domain.SourceSet, error) {
	if len(req.APIs) == 0 {
		return nil, errors.New("at least one --api is required")
	}
	if strings.TrimSpace(req.OutputDir) == "" {
		return nil, errors.New("output directory must not be empty")
	}

	src, err := g.sources.Load(ctx, req.APIs, req.RequirementsPath)
	if err != nil {
		return nil, fmt.Errorf("load sources: %w", err)
	}
	if len(src.APIs) == 0 {
		return nil, errors.New("openapi sources contain no operations")
	}

	if strings.TrimSpace(req.RequirementsPath) == "" {
		src.Requirements = nil
		src.Input = domain.InputOpenAPI
		return src, nil
	}
	if src.Requirements == nil || strings.TrimSpace(src.Requirements.Raw) == "" {
		return nil, fmt.Errorf("requirements file %q is empty; omit --req instead", req.RequirementsPath)
	}
	src.Input = domain.InputBoth
	return src, nil
}
