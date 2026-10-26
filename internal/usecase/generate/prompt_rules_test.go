package generate

import (
	"strings"
	"testing"
)

func TestJSONRuleDoesNotInviteStageAssumptions(t *testing.T) {
	if strings.Contains(jsonRule, "record it as an assumption") {
		t.Fatal("jsonRule must not invite assumptions in every stage")
	}
	if !strings.Contains(jsonRule, "never add extra keys") {
		t.Fatal("jsonRule must forbid extra keys")
	}
}

func TestSynthesisPromptsUseAllowedAPIKeys(t *testing.T) {
	for name, prompt := range map[string]string{
		"setup": setupSystemPrompt, "scenario": scenarioSystemPrompt,
	} {
		if !strings.Contains(prompt, `"key"`) || !strings.Contains(prompt, `"allowed_apis"`) {
			t.Errorf("%s prompt must tell the model to use the key field", name)
		}
	}
}

func TestOutlinePromptUsesPrefixValue(t *testing.T) {
	if !strings.Contains(outlineSystemPrompt, `value of "local_setup_prefix"`) {
		t.Fatal("outline prompt must refer to the value of local_setup_prefix")
	}
}

func TestSynthesisPromptsDescribeBodyWhenPresent(t *testing.T) {
	for _, prompt := range []string{setupSystemPrompt, scenarioSystemPrompt} {
		if !strings.Contains(prompt, "no request body") {
			t.Error("prompt must state that body is omitted when the API has no request body")
		}
	}
}
