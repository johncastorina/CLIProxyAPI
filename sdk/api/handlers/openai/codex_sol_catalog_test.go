package openai

import "testing"

func TestCodexClientCatalogUsesSolCapabilities(t *testing.T) {
	response := CodexClientModelsResponseForClient([]map[string]any{{"id": "gpt-6-sol"}}, "0.155.1", false)
	models, ok := response["models"].([]map[string]any)
	if !ok || len(models) != 1 {
		t.Fatalf("expected one Sol model, got %T with %d entries", response["models"], len(models))
	}
	model := models[0]
	for key, want := range map[string]any{
		"slug": "gpt-6-sol", "display_name": "GPT-6-Sol",
		"tool_mode": "code_mode_only", "use_responses_lite": true,
		"supports_reasoning_effort_updates": true,
	} {
		if model[key] != want {
			t.Errorf("%s = %v, want %v", key, model[key], want)
		}
	}
	if got := testIntModelValue(model, "context_window"); got != 272000 {
		t.Errorf("context_window = %d, want native Codex default 272000", got)
	}
	if got := testIntModelValue(model, "max_context_window"); got != 872000 {
		t.Errorf("max_context_window = %d, want native Codex maximum 872000", got)
	}
}
