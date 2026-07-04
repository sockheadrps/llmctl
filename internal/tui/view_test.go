package tui

import (
	"reflect"
	"testing"
)

func TestPickerSpinnerFrame(t *testing.T) {
	if got := pickerSpinnerFrame(0); got != "⠋" {
		t.Fatalf("expected first frame %q, got %q", "⠋", got)
	}
	if got := pickerSpinnerFrame(1); got != "⠙" {
		t.Fatalf("expected second frame %q, got %q", "⠙", got)
	}
	if got := pickerSpinnerFrame(11); got != "⠙" {
		t.Fatalf("expected looped frame %q, got %q", "⠙", got)
	}
}

func TestFormatDetailPairsUsesStackedLayoutWhenNarrow(t *testing.T) {
	got := formatDetailPairs([]detailPair{{label: "Port", value: "8080"}, {label: "Ctx Size", value: "4096"}}, 36)
	want := []string{"Port: 8080", "Ctx Size: 4096"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFormatDetailPairsUsesTwoColumnLayoutWhenWide(t *testing.T) {
	got := formatDetailPairs([]detailPair{{label: "Port", value: "8080"}, {label: "Ctx Size", value: "4096"}}, 90)
	want := []string{"Port: 8080    Ctx Size: 4096"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestSplitPaneHeightCapsDetailsPane(t *testing.T) {
	_, details := splitPaneHeight(40, 0)
	if details > 12 {
		t.Fatalf("expected details pane to stay compact, got %d", details)
	}
}
