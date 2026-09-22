package cliproxy

import (
	"context"
	"slices"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestSubscriptionWorkerModelsRouteThroughOAuth(t *testing.T) {
	for _, tc := range []struct {
		provider string
		plan     string
		model    string
	}{
		{"claude", "", "claude-opus-5-5"},
		{"codex", "pro", "gpt-6-sol"},
		{"codex", "plus", "gpt-6-sol"},
		{"codex", "team", "gpt-6-sol"},
	} {
		t.Run(tc.provider+"/"+tc.plan, func(t *testing.T) {
			id := "subscription-worker-" + tc.provider + "-" + tc.plan
			models := registry.GetGlobalRegistry()
			t.Cleanup(func() { models.UnregisterClient(id) })
			auth := &coreauth.Auth{
				ID: id, Provider: tc.provider, Status: coreauth.StatusActive,
				Attributes: map[string]string{"plan_type": tc.plan},
				Metadata:   map[string]any{"access_token": "synthetic-oauth-token"},
			}
			service := &Service{cfg: &config.Config{}}
			service.registerModelsForAuth(context.Background(), auth)
			var worker *registry.ModelInfo
			for _, model := range models.GetModelsForClient(id) {
				if model.ID == tc.model {
					worker = model
					break
				}
			}
			if worker == nil {
				t.Fatalf("OAuth account cannot route worker %q", tc.model)
			}
			if !slices.Contains(models.GetModelProviders(tc.model), tc.provider) {
				t.Fatalf("worker %q is not registered for provider %q", tc.model, tc.provider)
			}
		})
	}
}
