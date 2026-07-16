package statusserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryRetainsBoundedSnapshots(t *testing.T) {
	srv := NewServer()
	srv.historyLimit = 2

	srv.SetStatus(Status{Version: "one"})
	srv.SetStatus(Status{Version: "two"})
	srv.SetStatus(Status{Version: "three"})

	history := srv.History()
	if got, want := len(history), 2; got != want {
		t.Fatalf("history length = %d, want %d", got, want)
	}
	if got, want := history[0].Status.Version, "two"; got != want {
		t.Fatalf("oldest sample version = %q, want %q", got, want)
	}
	if got, want := history[1].Status.Version, "three"; got != want {
		t.Fatalf("newest sample version = %q, want %q", got, want)
	}
}

func TestHandlersServeDashboardAndHistory(t *testing.T) {
	srv := NewServer()
	srv.SetStatus(Status{
		Version: "v-test",
		Running: []RunningInfo{{
			Model:   "Qwen",
			Profile: "Default",
			Port:    8080,
			Health:  "up",
			TokS:    22.5,
		}},
	})

	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/status returned %d", resp.StatusCode)
	}
	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if got, want := status.Version, "v-test"; got != want {
		t.Fatalf("status version = %q, want %q", got, want)
	}

	resp, err = http.Get(ts.URL + "/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/history returned %d", resp.StatusCode)
	}
	var history History
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		t.Fatal(err)
	}
	if got, want := len(history.Samples), 1; got != want {
		t.Fatalf("history sample count = %d, want %d", got, want)
	}
	if got, want := history.Samples[0].Status.Running[0].Model, "Qwen"; got != want {
		t.Fatalf("history model = %q, want %q", got, want)
	}

	resp, err = http.Get(ts.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/dashboard returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	for _, want := range []string{"llmctl live dashboard", "/history", "/status"} {
		if !strings.Contains(html, want) {
			t.Fatalf("dashboard HTML missing %q", want)
		}
	}

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("/ returned %d, want redirect", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/dashboard" {
		t.Fatalf("redirect location = %q, want /dashboard", loc)
	}
}

func TestHandlersHideDashboardWhenDisabled(t *testing.T) {
	srv := NewServer()
	srv.ConfigureDashboard(false)
	ts := httptest.NewServer(srv.handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("/dashboard returned %d, want 404", resp.StatusCode)
	}

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err = client.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("/ returned %d, want redirect", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/status" {
		t.Fatalf("redirect location = %q, want /status", loc)
	}
}

func TestHistoryPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status_history.json")

	srv := NewServer()
	if err := srv.ConfigureHistoryPersistence(path, true); err != nil {
		t.Fatal(err)
	}
	srv.SetStatus(Status{Version: "one"})
	srv.SetStatus(Status{Version: "two"})

	restored := NewServer()
	if err := restored.ConfigureHistoryPersistence(path, true); err != nil {
		t.Fatal(err)
	}

	history := restored.History()
	if got, want := len(history), 2; got != want {
		t.Fatalf("restored history length = %d, want %d", got, want)
	}
	if got, want := history[0].Status.Version, "one"; got != want {
		t.Fatalf("restored first version = %q, want %q", got, want)
	}
	if got, want := history[1].Status.Version, "two"; got != want {
		t.Fatalf("restored second version = %q, want %q", got, want)
	}
}
