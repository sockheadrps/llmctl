package process

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sockheadrps/llmctl/internal/models"
)

func ptr[T any](v T) *T { return &v }

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name    string
		model   models.Model
		profile models.Profile
		want    []string
		absent  []string
	}{
		{
			name:    "local model uses --model flag",
			model:   models.Model{Path: "/models/llama.gguf"},
			profile: models.Profile{Port: 8080},
			want:    []string{"--model", "/models/llama.gguf", "--port", "8080"},
			absent:  []string{"-hf"},
		},
		{
			name:    "remote model uses -hf flag",
			model:   models.Model{HFRepo: "meta-llama/Llama-3-8B-Q4_K_M"},
			profile: models.Profile{Port: 8080},
			want:    []string{"-hf", "meta-llama/Llama-3-8B-Q4_K_M", "--port", "8080"},
			absent:  []string{"--model"},
		},
		{
			name:  "optional string fields omitted when empty",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
				Host: "",
			},
			absent: []string{"--host", "--alias"},
		},
		{
			name:  "host and alias included when set",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:  8080,
				Host:  "0.0.0.0",
				Alias: "mymodel",
			},
			want: []string{"--host", "0.0.0.0", "--alias", "mymodel"},
		},
		{
			name:  "ctx-size included when nonzero",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:    8080,
				CtxSize: 4096,
			},
			want:   []string{"--ctx-size", "4096"},
			absent: []string{},
		},
		{
			name:  "ctx-size omitted when zero",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:    8080,
				CtxSize: 0,
			},
			absent: []string{"--ctx-size"},
		},
		{
			name:  "sampling float pointers included when set",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:              8080,
				Temp:              ptr(0.7),
				TopP:              ptr(0.9),
				MinP:              ptr(0.05),
				PresencePenalty:   ptr(0.1),
				RepetitionPenalty: ptr(1.1),
				FrequencyPenalty:  ptr(0.0),
			},
			want: []string{
				"--temp", "0.7",
				"--top-p", "0.9",
				"--min-p", "0.05",
				"--presence-penalty", "0.1",
				"--repeat-penalty", "1.1",
				"--frequency-penalty", "0",
			},
		},
		{
			name:  "sampling pointers omitted when nil",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
			},
			absent: []string{"--temp", "--top-p", "--top-k", "--min-p"},
		},
		{
			name:  "flash-attn flag only added when true",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:      8080,
				FlashAttn: true,
			},
			want: []string{"--flash-attn", "on"},
		},
		{
			name:  "flash-attn omitted when false",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:      8080,
				FlashAttn: false,
			},
			absent: []string{"--flash-attn"},
		},
		{
			name:  "mmap true adds --mmap",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
				MMap: ptr(true),
			},
			want:   []string{"--mmap"},
			absent: []string{"--no-mmap"},
		},
		{
			name:  "mmap false adds --no-mmap",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
				MMap: ptr(false),
			},
			want:   []string{"--no-mmap"},
			absent: []string{"--mmap"},
		},
		{
			name:  "mmap nil omits both flags",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
				MMap: nil,
			},
			absent: []string{"--mmap", "--no-mmap"},
		},
		{
			name:  "gpu layers included when nonzero",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:      8080,
				GPULayers: 32,
			},
			want: []string{"--n-gpu-layers", "32"},
		},
		{
			name:  "gpu layers omitted when zero",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:      8080,
				GPULayers: 0,
			},
			absent: []string{"--n-gpu-layers"},
		},
		{
			name:  "reasoning fields included when set",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:            8080,
				Reasoning:       "deepseek",
				ReasoningBudget: ptr(2048),
				ReasoningFormat: "raw",
			},
			want: []string{"--reasoning", "deepseek", "--reasoning-budget", "2048", "--reasoning-format", "raw"},
		},
		{
			name:  "cache type flags included when set",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:       8080,
				CacheTypeK: "q8_0",
				CacheTypeV: "q4_0",
			},
			want: []string{"--cache-type-k", "q8_0", "--cache-type-v", "q4_0"},
		},
		{
			name:  "extra args appended at end",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port:      8080,
				ExtraArgs: []string{"--verbose", "--threads", "8"},
			},
			want: []string{"--verbose", "--threads", "8"},
		},
		{
			name:  "extra args empty produces no extra flags",
			model: models.Model{Path: "/m.gguf"},
			profile: models.Profile{
				Port: 8080,
			},
			want: []string{"--port", "8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildArgs(tt.model, tt.profile)
			for _, w := range tt.want {
				if !slices.Contains(got, w) {
					t.Errorf("expected %q in args %v", w, got)
				}
			}
			for _, a := range tt.absent {
				if slices.Contains(got, a) {
					t.Errorf("unexpected %q in args %v", a, got)
				}
			}
		})
	}
}

func TestBuildStartArgsNoVerboseByDefault(t *testing.T) {
	model := models.Model{Path: "/m.gguf"}
	profile := models.Profile{Port: 8080}

	got := buildStartArgs(model, profile, "")
	if slices.Contains(got, "-v") || slices.Contains(got, "--verbose") {
		t.Fatalf("expected -v not to be added automatically, got %v", got)
	}
	if slices.Contains(got, "--rpc") {
		t.Fatalf("expected non-RPC start args not to include --rpc, got %v", got)
	}
}

func TestBuildStartArgsAddsRPC(t *testing.T) {
	model := models.Model{Path: "/m.gguf"}
	profile := models.Profile{Port: 8080}

	got := buildStartArgs(model, profile, "127.0.0.1:50052")
	if !slices.Contains(got, "--rpc") {
		t.Fatalf("expected RPC start args to include --rpc, got %v", got)
	}
}

func TestBuildStartArgsPassesThroughVerboseFromExtraArgs(t *testing.T) {
	model := models.Model{Path: "/m.gguf"}
	profile := models.Profile{
		Port:      8080,
		ExtraArgs: []string{"--verbose"},
	}

	got := buildStartArgs(model, profile, "127.0.0.1:50052")
	if !slices.Contains(got, "--verbose") {
		t.Fatalf("expected --verbose from extra_args to be present, got %v", got)
	}
	count := 0
	for _, arg := range got {
		if arg == "-v" || arg == "--verbose" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one verbose flag, got %d in %v", count, got)
	}
}

func TestTailLog(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		f := filepath.Join(t.TempDir(), "server.log")
		if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return f
	}

	t.Run("returns last n lines when file has more", func(t *testing.T) {
		f := write(t, "line1\nline2\nline3\nline4\nline5\n")
		got, err := TailLog(f, 3)
		if err != nil {
			t.Fatal(err)
		}
		want := "line3\nline4\nline5"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns all lines when file has fewer than n", func(t *testing.T) {
		f := write(t, "a\nb\n")
		got, err := TailLog(f, 10)
		if err != nil {
			t.Fatal(err)
		}
		want := "a\nb"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns empty string for empty file", func(t *testing.T) {
		f := write(t, "")
		got, err := TailLog(f, 5)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := TailLog(filepath.Join(t.TempDir(), "nope.log"), 5)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestStartMissingExecutableReturnsConfigHintWithoutLog(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "missing.log")
	m := models.Model{Name: "Test Model", Path: filepath.Join(t.TempDir(), "model.gguf")}
	p := models.Profile{Name: "default", Port: 8080}

	_, err := Start("llmctl-definitely-missing-llama-server", m, p, logPath, "")
	if err == nil {
		t.Fatal("expected missing executable error")
	}

	msg := err.Error()
	for _, want := range []string{"llama-server binary", "llama_server_bin", "PATH"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}

	if _, statErr := os.Stat(logPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no log file for preflight failure, stat err = %v", statErr)
	}
}
