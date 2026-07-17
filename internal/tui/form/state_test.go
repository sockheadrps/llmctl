package form

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestDirty(t *testing.T) {
	if Dirty(false, false, false, false, false, false, 0, 0, []string{"a"}, []string{"a"}, nil, nil) {
		t.Fatal("expected identical form state to be clean")
	}
	if !Dirty(true, false, false, false, false, false, 0, 0, []string{"a"}, []string{"a"}, nil, nil) {
		t.Fatal("expected flash mismatch to be dirty")
	}
}

func TestEnsureVisible(t *testing.T) {
	if got := EnsureVisible(0, 0, 3, 5); got != 0 {
		t.Fatalf("expected no scroll, got %d", got)
	}
	if got := EnsureVisible(4, 0, 3, 5); got != 2 {
		t.Fatalf("expected scroll to 2, got %d", got)
	}
}

func TestAdvanceAutoScroll(t *testing.T) {
	offset, dir, pause := AdvanceAutoScroll(3, -1, 1, 2, 5, 2)
	if offset != 0 || dir != 1 || pause != 0 {
		t.Fatalf("expected reset scroll state, got offset %d dir %d pause %d", offset, dir, pause)
	}
}

func TestDescriptionWindow(t *testing.T) {
	got := DescriptionWindow([]string{"one", "two", "three", "four"}, 3, 1)
	want := []string{"two", "three", "four"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFocusedFlagAndFlagInputSync(t *testing.T) {
	if got := FocusedFlag(FieldKey, len(Labels)); got != "" {
		t.Fatalf("expected blank flag for non-mapped field, got %q", got)
	}
	if got := FocusedFlag(len(Labels), len(Labels)); got != "--flash-attn" {
		t.Fatalf("expected flash-attn flag, got %q", got)
	}

	input := textinput.New()
	input = SyncFlagInput(input, len(Labels), len(Labels), map[string]string{
		"--flash-attn": "off",
	})
	if got := input.Value(); got != "off" {
		t.Fatalf("expected override value off, got %q", got)
	}
}

func TestCommitFlagInput(t *testing.T) {
	input := textinput.New()
	input.SetValue("false")
	got := CommitFlagInput(len(Labels), input, nil)
	if got["--flash-attn"] != "false" {
		t.Fatalf("expected flash override false, got %#v", got)
	}
}
