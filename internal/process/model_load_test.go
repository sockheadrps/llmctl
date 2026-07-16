package process

import (
	"os"
	"testing"
)

func TestParseModelLoadSlicesAggregatesBuffers(t *testing.T) {
	raw := `
load_tensors: CUDA0 model buffer size = 4660.00 MiB
llama_kv_cache: CUDA0 KV buffer size = 512.00 MiB
sched_reserve: CUDA0 compute buffer size = 128.00 MiB
load_tensors: RPC0 model buffer size  = 11730.00 MiB
llama_kv_cache: RPC0 KV buffer size = 256.00 MiB
sched_reserve: RPC0 compute buffer size = 64.00 MiB
CPU_Mapped model buffer size = 682.00 MiB
`

	got, err := ParseModelLoadSlices(writeTempLog(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 model slices, got %d", len(got))
	}
	if got[0].Name != "CUDA 0" || got[0].UsedMiB != 5300 {
		t.Fatalf("unexpected first slice: %+v", got[0])
	}
	if got[1].Name != "RPC 0" || got[1].UsedMiB != 12050 {
		t.Fatalf("unexpected second slice: %+v", got[1])
	}
}

func writeTempLog(t *testing.T, raw string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.log")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(raw); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}
