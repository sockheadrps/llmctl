package util

import (
	"fmt"
	"net"
)

// LANAddrs returns the non-loopback, non-link-local IPv4 addresses assigned
// to local interfaces — the addresses reachable by other machines on the LAN.
func LANAddrs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var addrs []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range ifAddrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ip4 := ip.To4()
			if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			addrs = append(addrs, ip4.String())
		}
	}
	return addrs
}

// StatusServerAddrs returns "ip:port" strings for each LAN address, ready to
// paste into a remote llmctl's Remote Status Address field.
func StatusServerAddrs(port int) []string {
	addrs := LANAddrs()
	out := make([]string, len(addrs))
	for i, ip := range addrs {
		out[i] = fmt.Sprintf("%s:%d", ip, port)
	}
	return out
}
