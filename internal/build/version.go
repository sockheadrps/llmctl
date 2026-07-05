// Package build exposes build-time metadata injected via ldflags.
package build

// Version is the release tag. Set at build time with:
//
//	go build -ldflags "-X github.com/sockheadrps/llmctl/internal/build.Version=v0.2.0" .
var Version = "v0.1.6"
