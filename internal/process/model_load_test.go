package process

import "testing"

func TestParseModelLoadSlices(t *testing.T) {
	raw := `
CUDA0 model buffer size = 4660.00 MiB
RPC0 model buffer size  = 11730.00 MiB
CPU_Mapped model buffer size = 682.00 MiB
`

	got := parseModelLoadSlices(raw)
	if len(got) != 2 {
		t.Fatalf("expected 2 model slices, got %d", len(got))
	}
	if got[0].Name != "CUDA0" || got[0].UsedMiB != 4660 {
		t.Fatalf("unexpected first slice: %+v", got[0])
	}
	if got[1].Name != "RPC0" || got[1].UsedMiB != 11730 {
		t.Fatalf("unexpected second slice: %+v", got[1])
	}
}
