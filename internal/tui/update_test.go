package tui

import (
	"reflect"
	"testing"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

func TestModelCursorTargetsIncludeAddModel(t *testing.T) {
	rows := []row{
		{kind: rowModel, modelKey: "alpha", label: "Alpha"},
		{kind: rowProfile, modelKey: "alpha", profileKey: "default", label: "Default"},
		{kind: rowAddProfile, modelKey: "alpha", label: "+ New Profile"},
		{kind: rowModel, modelKey: "beta", label: "Beta"},
		{kind: rowProfile, modelKey: "beta", profileKey: "default", label: "Default"},
		{kind: rowAddModel, label: "+ Add Model"},
	}

	got := modelCursorTargets(rows)
	want := []int{0, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestEnterModelsPaneExpandsFirstModel(t *testing.T) {
	m := Model{
		cfg: &config.Config{Models: map[string]models.Model{
			"alpha": {Name: "Alpha", Profiles: map[string]models.Profile{"default": {Name: "Default"}}},
		}},
	}

	m.enterModelsPane()

	if m.expandedModelKey != "alpha" {
		t.Fatalf("expected first model to be expanded, got %q", m.expandedModelKey)
	}
	if m.cursor != 0 {
		t.Fatalf("expected cursor to land on the first model row, got %d", m.cursor)
	}
}
