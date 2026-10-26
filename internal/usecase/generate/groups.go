package generate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/nguyentin05/spectest/internal/domain"
)

// apiGroup is the unit of parallel outline: all APIs of one server that share
// a resource segment.
type apiGroup struct {
	ID       string
	Server   string
	Resource string
	APIs     []domain.API
}

var (
	versionSegRe = regexp.MustCompile(`^v[0-9]+(\.[0-9]+)?$`)
	nonAlnumRe   = regexp.MustCompile(`[^a-z0-9]+`)
)

func slugify(s string) string {
	return strings.Trim(nonAlnumRe.ReplaceAllString(strings.ToLower(s), "_"), "_")
}

func isGenericSegment(s string) bool {
	s = strings.ToLower(s)
	return s == "api" || s == "rest" || versionSegRe.MatchString(s)
}

// resourceSegment derives the resource name from a path: skip generic leading
// segments (api, rest, vN), then take the first static segment.
func resourceSegment(path string) string {
	segs := splitCallPath(path)
	i := 0
	for i < len(segs) && isGenericSegment(segs[i]) {
		i++
	}
	for ; i < len(segs); i++ {
		if segs[i] == "" || isTemplateSegment(segs[i]) {
			continue
		}
		if s := slugify(segs[i]); s != "" {
			return s
		}
	}
	return "root"
}

// buildGroups groups APIs by (server, resource). Order is deterministic:
// servers keep their --api order, then resource, path, method.
func buildGroups(src *domain.SourceSet) []apiGroup {
	serverIdx := make(map[string]int, len(src.Servers))
	for i, s := range src.Servers {
		serverIdx[s] = i
	}

	type item struct {
		api domain.API
		res string
	}
	items := make([]item, 0, len(src.APIs))
	for _, a := range src.APIs {
		items = append(items, item{api: a, res: resourceSegment(a.Path)})
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if si, sj := serverIdx[a.api.Server], serverIdx[b.api.Server]; si != sj {
			return si < sj
		}
		if a.res != b.res {
			return a.res < b.res
		}
		if a.api.Path != b.api.Path {
			return a.api.Path < b.api.Path
		}
		return a.api.Method < b.api.Method
	})

	var groups []apiGroup
	index := map[string]int{}
	taken := map[string]struct{}{}
	for _, it := range items {
		k := it.api.Server + "\x00" + it.res
		gi, ok := index[k]
		if !ok {
			base := slugify(it.api.Server + "_" + it.res)
			id := base
			for n := 2; ; n++ {
				if _, clash := taken[id]; !clash {
					break
				}
				id = fmt.Sprintf("%s_%d", base, n)
			}
			taken[id] = struct{}{}
			groups = append(groups, apiGroup{ID: id, Server: it.api.Server, Resource: it.res})
			gi = len(groups) - 1
			index[k] = gi
		}
		groups[gi].APIs = append(groups[gi].APIs, it.api)
	}
	return groups
}

// setupOwner returns the ID of the group that owns a local setup name: the
// longest group ID g such that name starts with g+"_". Empty means "common".
func setupOwner(name string, groups []apiGroup) string {
	owner := ""
	for _, g := range groups {
		if strings.HasPrefix(name, g.ID+"_") && len(g.ID) > len(owner) {
			owner = g.ID
		}
	}
	return owner
}

// groupOfAPIs maps every API key to the ID of its group.
func groupOfAPIs(groups []apiGroup) map[string]string {
	out := map[string]string{}
	for _, g := range groups {
		for _, a := range g.APIs {
			out[a.Key] = g.ID
		}
	}
	return out
}
