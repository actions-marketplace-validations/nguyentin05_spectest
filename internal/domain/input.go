package domain

// InputMode identifies where the API list of a run came from.
// v3 supports exactly two modes: OpenAPI-only and OpenAPI-plus-requirements.
type InputMode string

const (
	InputOpenAPI InputMode = "openapi"
	InputBoth    InputMode = "both"
)

// Valid reports whether m is a recognized input mode.
func (m InputMode) Valid() bool {
	switch m {
	case InputOpenAPI, InputBoth:
		return true
	}
	return false
}
