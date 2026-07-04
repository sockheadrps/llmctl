package tui

import "testing"

func TestFormLabelsIncludeExtendedProfileOptions(t *testing.T) {
	contains := func(labels []string, want string) bool {
		for _, label := range labels {
			if label == want {
				return true
			}
		}
		return false
	}

	for _, want := range []string{"Host", "Batch Size", "Parallel Slots", "Reasoning"} {
		if !contains(formLabels, want) {
			t.Fatalf("expected form labels to include %q", want)
		}
	}
}

func TestFieldDescriptionForKnownParam(t *testing.T) {
	if got := formFieldDescription(fieldHost); got == "" {
		t.Fatal("expected a description for host")
	}
	if got := formFieldDescription(fieldReasoning); got == "" {
		t.Fatal("expected a description for reasoning")
	}
}
