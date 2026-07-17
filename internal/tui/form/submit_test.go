package form

import (
	"testing"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

func TestBuildProfileSubmissionBuildsProfile(t *testing.T) {
	values := make([]string, len(Labels))
	values[FieldKey] = "profile-a"
	values[FieldHost] = "127.0.0.1"
	values[FieldPort] = "8123"
	values[FieldGPULayers] = "5"

	got, err := BuildProfileSubmission(values, false, "", nil, true, false, false, 2, true, map[string]string{"--port": "--p"})
	if err != nil {
		t.Fatalf("BuildProfileSubmission returned error: %v", err)
	}
	if got.Key != "profile-a" {
		t.Fatalf("expected key profile-a, got %q", got.Key)
	}
	if got.Profile.Port != 8123 {
		t.Fatalf("expected port 8123, got %d", got.Profile.Port)
	}
	if got.Profile.TensorSplit != "2,3" {
		t.Fatalf("expected tensor split 2,3, got %q", got.Profile.TensorSplit)
	}
	if got.Profile.RPCEnabled != nil {
		t.Fatalf("expected nil RPC override, got %#v", got.Profile.RPCEnabled)
	}
	if got.Profile.FlagOverrides["--port"] != "--p" {
		t.Fatalf("expected flag override copy, got %#v", got.Profile.FlagOverrides)
	}
}

func TestBuildProfileSubmissionRejectsDuplicateKey(t *testing.T) {
	values := make([]string, len(Labels))
	values[FieldKey] = "profile-a"
	values[FieldPort] = "8123"

	_, err := BuildProfileSubmission(values, false, "", map[string]models.Profile{"profile-a": {}}, false, false, false, 0, false, nil)
	if err == nil {
		t.Fatal("expected duplicate profile key to fail")
	}
}

func TestCommitProfileSubmissionWritesProfile(t *testing.T) {
	cfg := &config.Config{Models: map[string]models.Model{
		"model-a": {Profiles: map[string]models.Profile{"old": {Name: "old"}}},
	}}
	CommitProfileSubmission(cfg, "model-a", "old", true, SubmitResult{
		Key:     "new",
		Profile: models.Profile{Name: "new", Port: 8123},
	})

	mdl := cfg.Models["model-a"]
	if _, ok := mdl.Profiles["old"]; ok {
		t.Fatal("expected old profile to be removed after rename")
	}
	if got := mdl.Profiles["new"]; got.Port != 8123 {
		t.Fatalf("expected new profile to be written, got %#v", got)
	}
}
