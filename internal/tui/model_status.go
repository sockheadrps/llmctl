package tui

import (
	"os"
	"strings"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	"github.com/sockheadrps/llmctl/internal/util"
)

// status-server only
func (m Model) shouldRunStatusServer() bool {
	if m.cfg == nil {
		return false
	}
	return m.cfg.StatusServerEnabled
}

// status-server only
func (m Model) statusServerBindAddr() (string, int) {
	if m.cfg == nil {
		return "0.0.0.0", 11435
	}
	host := m.cfg.StatusServerHost
	if host == "" {
		host = "0.0.0.0"
	}
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	return host, port
}

// status-server only
func (m *Model) reconcileStatusServer() error {
	if !m.shouldRunStatusServer() {
		if m.statusServer != nil {
			m.statusServer.Stop()
			m.statusServer = nil
			m.statusServerHost = ""
			m.statusServerPort = 0
		}
		return nil
	}

	host, port := m.statusServerBindAddr()
	if m.statusServer != nil && m.statusServerHost == host && m.statusServerPort == port {
		historyPath, err := util.StatusHistoryFile()
		if err != nil {
			return err
		}
		if err := m.statusServer.ConfigureHistoryPersistence(historyPath, m.cfg.StatusHistoryPersistEnabled()); err != nil {
			return err
		}
		m.statusServer.ConfigureDashboard(m.cfg.StatusDashboardEnabled())
		return nil
	}
	if m.statusServer != nil {
		m.statusServer.Stop()
		m.statusServer = nil
		m.statusServerHost = ""
		m.statusServerPort = 0
	}

	srv := statusserver.NewServer()
	if err := srv.Start(host, port); err != nil {
		return err
	}
	historyPath, err := util.StatusHistoryFile()
	if err != nil {
		srv.Stop()
		return err
	}
	if err := srv.ConfigureHistoryPersistence(historyPath, m.cfg.StatusHistoryPersistEnabled()); err != nil {
		srv.Stop()
		return err
	}
	srv.ConfigureDashboard(m.cfg.StatusDashboardEnabled())
	m.statusServer = srv
	m.statusServerHost = host
	m.statusServerPort = port
	m.pushStatusServer()
	return nil
}

// client-mode only
func (m *Model) reconcileStatusPublisher() {
	if m.statusPublisher == nil {
		return
	}
	if m.cfg == nil || !m.cfg.RPCEnabled || m.cfg.RPCMode != "client" || strings.TrimSpace(m.cfg.RemoteStatusAddr) == "" {
		m.statusPublisher.Stop()
		m.statusPublisherAddr = ""
		return
	}
	addr := strings.TrimSpace(m.cfg.RemoteStatusAddr)
	if m.statusPublisherAddr == addr {
		return
	}
	m.statusPublisher.Start(addr)
	m.statusPublisherAddr = addr
	m.statusPublisher.Update(m.buildStatusSnapshot())
}

// client-mode only
func clientID() string {
	if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return "llmctl-client"
}

// client-mode only
func clientName() string {
	return clientID()
}

// shared
// pushStatusServer updates the local status server snapshot with current state.
// In RPC client mode it also publishes the snapshot to the remote status server.
func (m *Model) pushStatusServer() {
	st := m.buildStatusSnapshot()
	if m.statusServer != nil {
		m.statusServer.SetStatus(st)
	}
	if m.statusPublisher != nil {
		m.statusPublisher.Update(st)
	}
}

// shared
// buildStatusSnapshot assembles a statusserver.Status from current model state.
func (m *Model) buildStatusSnapshot() statusserver.Status {
	toGPUDeviceInfo := func(device gpu.DeviceUsage) statusserver.GPUDeviceInfo {
		info := statusserver.GPUDeviceInfo{
			Index:    device.Index,
			UUID:     device.UUID,
			Name:     device.Name,
			UsedMiB:  device.UsedMiB,
			TotalMiB: device.TotalMiB,
		}
		if info.Name == "" {
			info.Name = "Unknown GPU"
		}
		return info
	}

	running := make([]statusserver.RunningInfo, 0, len(m.running))
	for _, r := range m.running {
		key := r.ModelKey + "/" + r.ProfileKey
		h := m.health[key]
		if h == "" || m.pendingInstances[key] {
			h = health.StatusLoading
		}
		peakVal := m.tokPeak[key]
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
			if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.MaxTokPerSec > peakVal {
				peakVal = p.MaxTokPerSec
			}
		}
		info := statusserver.RunningInfo{
			Model:   r.ModelName,
			Profile: r.ProfileName,
			Port:    r.Port,
			Health:  string(h),
			TokS:    m.tokRates[key],
			TokPeak: peakVal,
		}
		if hist := m.tokHistory[key]; len(hist) > 0 {
			var sum float64
			for _, v := range hist {
				sum += v
			}
			info.TokAvg = sum / float64(len(hist))
			const maxSend = 20
			if len(hist) > maxSend {
				info.TokHistory = hist[len(hist)-maxSend:]
			} else {
				info.TokHistory = hist
			}
		} else if histAvg, ok := m.tokRateHistory.average(key); ok && histAvg > 0 {
			info.TokAvg = histAvg
		}
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
			if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.Alias != "" {
				info.Alias = p.Alias
			}
		}
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok && mdl.Path != "" {
			if stat, err := os.Stat(mdl.Path); err == nil {
				info.ModelSizeBytes = stat.Size()
			}
		}
		cpuOnly := false
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
			if p, ok := mdl.Profiles[r.ProfileKey]; ok {
				cpuOnly = p.CPUOnly
			}
		}
		if !cpuOnly && h == health.StatusUp {
			if slices, err := m.modelLoadSlices(r.LogFile); err == nil {
				info.GPUs = slices
				for _, slice := range slices {
					info.VRAMMiB += slice.UsedMiB
				}
			}
		} else {
			if mb, ok := m.ramByPID[r.PID]; ok {
				info.RAMMiB = mb
			}
		}
		running = append(running, info)
	}

	st := statusserver.Status{
		Version: build.Version,
		Running: running,
	}

	if m.cfg.RPCEnabled && m.cfg.RPCMode == "server" && m.rpcServerAlive {
		rpcInfo := &statusserver.RPCInfo{
			Up:   true,
			Host: m.cfg.RPCServerHost,
			Port: m.cfg.RPCServerPort,
		}
		st.RPCServer = rpcInfo
	}

	if m.gpuAvailable && m.gpuUsage.TotalMiB > 0 {
		devices := make([]statusserver.GPUDeviceInfo, 0, len(m.gpuDevices))
		for _, device := range m.gpuDevices {
			devices = append(devices, toGPUDeviceInfo(device))
		}
		st.GPU = &statusserver.GPUInfo{
			Name:     m.gpuName,
			TotalMiB: m.gpuUsage.TotalMiB,
			UsedMiB:  m.gpuUsage.UsedMiB,
			Devices:  devices,
		}
	}

	return st
}
