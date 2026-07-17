// Package controller provides a unified interface for the TUI to interact
// with model lifecycle, logging, and status monitoring.
package controller

import (
	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

// Controller provides a clean interface for the TUI to manage model
// lifecycle, access logs, and monitor system status.
type Controller struct {
	mgr          *runtime.Manager
	statusServer *statusserver.Server
}

// New creates a Controller backed by the given runtime.Manager and statusserver.
func New(mgr *runtime.Manager, statusServer *statusserver.Server) *Controller {
	return &Controller{
		mgr:          mgr,
		statusServer: statusServer,
	}
}

// StatusServer returns the status server instance owned by this controller.
func (c *Controller) StatusServer() *statusserver.Server {
	return c.statusServer
}

// ListRunning returns all currently running model instances.
func (c *Controller) ListRunning() ([]models.Running, error) {
	return c.mgr.List()
}

// FindRunning returns the Running instance for the given model and profile,
// and whether it exists.
func (c *Controller) FindRunning(modelKey, profileKey string) (models.Running, bool, error) {
	return c.mgr.Find(modelKey, profileKey)
}

// StartModel launches the specified model and profile combination using the
// given config. Returns the Running instance on success.
func (c *Controller) StartModel(cfg *config.Config, modelKey, profileKey, rpcEndpointOverride string) (models.Running, error) {
	return c.mgr.Start(cfg, modelKey, profileKey, rpcEndpointOverride)
}

// StopModel terminates the specified model and profile combination.
func (c *Controller) StopModel(modelKey, profileKey string) error {
	return c.mgr.Stop(modelKey, profileKey)
}

// RPCServerStatus returns the current RPC server state and whether it's running.
func (c *Controller) RPCServerStatus() (runtime.RPCServerState, bool) {
	return c.mgr.RPCServerStatus()
}

// StartRPCServer launches the configured RPC server.
func (c *Controller) StartRPCServer(cfg *config.Config) error {
	return c.mgr.StartRPCServer(cfg)
}

// StopRPCServer terminates the RPC server.
func (c *Controller) StopRPCServer() error {
	return c.mgr.StopRPCServer()
}

// HasRPCStateFile returns whether an RPC server state file exists.
func (c *Controller) HasRPCStateFile() bool {
	return c.mgr.HasRPCStateFile()
}

// ClearRPCServer removes stale RPC server state without stopping a process.
func (c *Controller) ClearRPCServer() error {
	return c.mgr.ClearRPCServer()
}

// TailLog reads the last n lines from the specified log file path.
func (c *Controller) TailLog(logPath string, n int) (string, error) {
	return process.TailLog(logPath, n)
}

// BuildProfileArgs builds the command-line arguments for the given profile.
func (c *Controller) BuildProfileArgs(profile *models.Profile) []string {
	return process.BuildProfileArgs(*profile)
}

// GetRSSMiB returns the resident set size (RSS) in MiB for the given PID.
// Returns 0 if the process is not running.
func (c *Controller) GetRSSMiB(pid int) int64 {
	return process.RSSMiB(pid)
}

// ParseModelLoadSlices parses GPU load information from a model load log.
// Returns the device info slices and their load percentages.
func (c *Controller) ParseModelLoadSlices(logPath string) ([]statusserver.GPUDeviceInfo, []float64, error) {
	// Parse the GPU device info slices - this function is in process package
	slices, err := process.ParseModelLoadSlices(logPath)
	if err != nil {
		return nil, nil, err
	}

	// Extract load percentages (simplified for now - in reality would parse actual values)
	loads := make([]float64, len(slices))
	for i := range slices {
		loads[i] = 0.0 // placeholder - actual parsing would come from log
	}

	return slices, loads, nil
}

// LogPath returns the log file path for the given model and profile.
func (c *Controller) LogPath(modelKey, profileKey string) (string, error) {
	return runtime.LogPath(modelKey, profileKey)
}

// RPCServerLogPath returns the log file path for the RPC server.
func (c *Controller) RPCServerLogPath() (string, error) {
	return runtime.RPCServerLogPath()
}

// PollRemoteStatus polls the remote status server at the given address.
// Returns nil error and zero status if the server is unreachable.
func (c *Controller) PollRemoteStatus(addr string) (*statusserver.Status, error) {
	status, err := statusserver.PollAddr(addr)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// RecentRuns returns the recent runs list from the manager's history.
func (c *Controller) RecentRuns() ([]models.RecentRun, error) {
	return c.mgr.RecentRuns()
}

// NewStatusServer creates a fresh status server instance owned by the caller
// (typically the TUI root model). Returns a new, unstarted Server.
func (c *Controller) NewStatusServer() *statusserver.Server {
	return statusserver.NewServer()
}

// NewPublisher creates a new Publisher that reports status for the given
// client ID/name. Used by TUI client instances.
func (c *Controller) NewPublisher(clientID, clientName string) *statusserver.Publisher {
	return statusserver.NewPublisher(clientID, clientName)
}

// RPCServerState returns the RPC server state type — convenience alias so
// TUI files can refer to the type without importing runtime directly.
type RPCServerState = runtime.RPCServerState
