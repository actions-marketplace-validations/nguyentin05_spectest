package generate

import (
	"strings"
	"testing"
)

func TestSavePromptsUseNameToPath(t *testing.T) {
	sc := &synthCtx{plan: &fullPlan{}}

	setupSystem, _, err := buildSetupPrompt(sc, planSetup{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(setupSystem, `"body.x": "name"`) {
		t.Fatal("setup prompt still teaches reversed save format")
	}
	if !strings.Contains(setupSystem, `"order_id": "body.id"`) {
		t.Fatal("setup prompt missing correct save example")
	}

	scenarioSystem, _, err := buildScenarioBatchPrompt(sc, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(scenarioSystem, `"save"`) || !strings.Contains(scenarioSystem, `"order_id": "body.id"`) {
		t.Fatal("scenario prompt missing correct save rule")
	}
}
