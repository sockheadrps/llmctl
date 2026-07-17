package form

import "testing"

func TestParseProfileArgs(t *testing.T) {
	got, extra := ParseProfileArgs(
		"llama-server --port 8123 --flash-attn off --verbose",
		4,
		func(i int) string {
			if i == 1 {
				return "--port"
			}
			return ""
		},
		4,
		3,
	)

	if got[1] != "8123" {
		t.Fatalf("expected port 8123, got %q", got[1])
	}
	if got[4] != "off" {
		t.Fatalf("expected flash toggle off at index 4, got %q", got[4])
	}
	if got[3] != "--verbose" {
		t.Fatalf("expected extra args at index 3, got %q", got[3])
	}
	if len(extra) != 1 || extra[0] != "--verbose" {
		t.Fatalf("expected one extra arg, got %#v", extra)
	}
}

func TestParseHelpers(t *testing.T) {
	if got, err := ParseIntOrZero(""); err != nil || got != 0 {
		t.Fatalf("ParseIntOrZero blank = %d, %v", got, err)
	}

	b, err := ParseBoolPtr("true")
	if err != nil || b == nil || !*b {
		t.Fatalf("ParseBoolPtr true = %#v, %v", b, err)
	}

	if got, err := ParseReasoning(" AUTO "); err != nil || got != "auto" {
		t.Fatalf("ParseReasoning auto = %q, %v", got, err)
	}
}
