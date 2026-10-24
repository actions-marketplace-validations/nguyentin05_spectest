package domain

// TrackFile is the `spectest.track.yaml` fingerprint of the last processed
// OpenAPI and requirements inputs. It is overwritten on every run.
type TrackFile struct {
	LastUpdate int                 `yaml:"last_update" json:"last_update"`
	Sources    Sources             `yaml:"sources" json:"sources"`
	APIs       map[string]APIState `yaml:"apis" json:"apis"`
}

// Sources records the input files and content hashes seen on the last run.
// Removed servers are dropped from OpenAPI. Requirements is nil when --req is absent.
type Sources struct {
	OpenAPI      map[string]FileState `yaml:"openapi,omitempty" json:"openapi,omitempty"`
	Requirements *RequirementState    `yaml:"requirements,omitempty" json:"requirements,omitempty"`
}

// FileState is a source file path together with its raw-content hash.
type FileState struct {
	File string `yaml:"file" json:"file"`
	Hash string `yaml:"hash" json:"hash"`
}

// RequirementState stores the requirements file, normalized hash, and raw text.
type RequirementState struct {
	File string `yaml:"file" json:"file"`
	Hash string `yaml:"hash" json:"hash"`
	Raw  string `yaml:"raw" json:"raw"`
}

// APIState is the tracked state of one API key. Removed APIs are retained
// forever so they can be detected as restored later.
type APIState struct {
	Hash   string       `yaml:"hash" json:"hash"`
	Status string       `yaml:"status" json:"status"`
	Sig    APISignature `yaml:"sig" json:"sig"`
}

// APISignature is a compact, human- and LLM-readable summary of an operation.
// It is used for diffs and prompts but never participates in hashing.
type APISignature struct {
	Request   map[string]string `yaml:"request" json:"request"`
	Responses []int             `yaml:"responses" json:"responses"`
	Auth      []string          `yaml:"auth" json:"auth"`
}
