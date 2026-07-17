package form

import (
	"testing"

	"github.com/sockheadrps/llmctl/internal/models"
)

func TestBuildNewProfileDefaults(t *testing.T) {
	got := BuildNewProfileDefaults(8123, map[int]string{
		FieldHost: "0.0.0.0",
		FieldMMap: "false",
	})

	if got[FieldPort] != "8123" {
		t.Fatalf("expected port 8123, got %q", got[FieldPort])
	}
	if got[FieldHost] != "0.0.0.0" {
		t.Fatalf("expected host override, got %q", got[FieldHost])
	}
	if got[FieldMMap] != "false" {
		t.Fatalf("expected mmap override, got %q", got[FieldMMap])
	}
}

func TestBuildEditProfileDefaults(t *testing.T) {
	got, layers := BuildEditProfileDefaults("profile-a", models.Profile{
		Host:        "127.0.0.1",
		Port:        9000,
		TensorSplit: "3,5",
	})

	if got[FieldKey] != "profile-a" {
		t.Fatalf("expected key profile-a, got %q", got[FieldKey])
	}
	if got[FieldPort] != "9000" {
		t.Fatalf("expected port 9000, got %q", got[FieldPort])
	}
	if layers != 3 {
		t.Fatalf("expected client layers 3, got %d", layers)
	}
}
