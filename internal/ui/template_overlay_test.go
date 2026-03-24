package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

func TestRenderTemplateOverlayShowsFilterInput(t *testing.T) {
	templates := []model.ResourceTemplate{
		{Name: "Deployment", Description: "Create a Deployment", Category: "Workloads"},
		{Name: "Service", Description: "Create a Service", Category: "Networking"},
	}
	result := RenderTemplateOverlay(templates, "dep", 0, true, 25)
	assert.Contains(t, result, "filter>")
	assert.Contains(t, result, "dep")
}

func TestRenderTemplateOverlayShowsFilterLabel(t *testing.T) {
	templates := []model.ResourceTemplate{
		{Name: "Deployment", Description: "Create a Deployment", Category: "Workloads"},
	}
	result := RenderTemplateOverlay(templates, "dep", 0, false, 25)
	assert.Contains(t, result, "filter:")
	assert.Contains(t, result, "dep")
}

func TestRenderTemplateOverlayNoFilterShowsSlashHint(t *testing.T) {
	templates := []model.ResourceTemplate{
		{Name: "Deployment", Description: "Create a Deployment", Category: "Workloads"},
	}
	result := RenderTemplateOverlay(templates, "", 0, false, 25)
	assert.Contains(t, result, "/: filter")
}

func TestRenderTemplateOverlayEmptyTemplates(t *testing.T) {
	result := RenderTemplateOverlay(nil, "", 0, false, 25)
	assert.Contains(t, result, "No templates available")
}

func TestRenderTemplateOverlayNoMatchingTemplates(t *testing.T) {
	templates := []model.ResourceTemplate{
		{Name: "Deployment", Description: "Create a Deployment", Category: "Workloads"},
	}
	// Pass empty slice (caller filters before passing).
	result := RenderTemplateOverlay(templates[:0], "xyz", 0, false, 25)
	assert.Contains(t, result, "No templates available")
}
