package models

import (
	"testing"
)

func TestDefaultProfileHasSensibleValues(t *testing.T) {
	p := DefaultProfile()
	if p.Name != "default" {
		t.Errorf("Name = %q; want %q", p.Name, "default")
	}
	if p.CtxSize != 8192 {
		t.Errorf("CtxSize = %d; want %d", p.CtxSize, 8192)
	}
	if p.Temp == nil || *p.Temp != 0.6 {
		t.Errorf("Temp = %v; want 0.6", p.Temp)
	}
	if p.TopP == nil || *p.TopP != 0.95 {
		t.Errorf("TopP = %v; want 0.95", p.TopP)
	}
	if !p.FlashAttn {
		t.Errorf("FlashAttn should default to true")
	}
	if p.GPULayers != 999 {
		t.Errorf("GPULayers = %d; want 999", p.GPULayers)
	}
}

func TestDefaultProfilePortCanBeOverridden(t *testing.T) {
	p := DefaultProfile()
	if p.Port != 8080 {
		t.Errorf("default Port = %d; want 8080 (caller overrides)", p.Port)
	}
	// Caller pattern: use SuggestPort result to override.
	p.Port = 9090
	if p.Port != 9090 {
		t.Error("Port override didn't stick")
	}
}

func TestSuggestPortAboveUsed(t *testing.T) {
	got := SuggestPort([]int{8080, 8081, 8090})
	if got <= 8090 {
		t.Errorf("SuggestPort([8080,8081,8090]) = %d; want > 8090", got)
	}
}

func TestSuggestPortEmptyUsesBase(t *testing.T) {
	got := SuggestPort(nil)
	if got < 8080 {
		t.Errorf("SuggestPort(nil) = %d; want >= 8080", got)
	}
}
