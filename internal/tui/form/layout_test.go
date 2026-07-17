package form

import "testing"

func TestAdjustTensorSplitClampsToBounds(t *testing.T) {
	if got := AdjustTensorSplit(0, 5, -1); got != 0 {
		t.Fatalf("expected lower clamp at 0, got %d", got)
	}
	if got := AdjustTensorSplit(4, 5, 5); got != 5 {
		t.Fatalf("expected upper clamp at 5, got %d", got)
	}
	if got := AdjustTensorSplit(2, 0, 3); got != 5 {
		t.Fatalf("expected unbounded growth when total is zero, got %d", got)
	}
}
