package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultNetworkInternetConn = "Wired connection 1"
	defaultNetworkRPCConn      = "enp8s0"
	defaultNetworkIface        = "enp8s0"
)

var (
	networkInternetConn string
	networkRPCConn      string
	networkIface        string
)

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Switch or inspect network profiles",
	Long: `network switches between the internet and RPC Ethernet setups used for
local host networking, and shows the current link state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var networkRpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "Switch to the RPC network profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runNetworkDown(networkInternetConn); err != nil {
			return err
		}
		if err := runNetworkUp(networkRPCConn); err != nil {
			return err
		}
		fmt.Printf("switched to RPC network (%s)\n", networkRPCConn)
		return nil
	},
}

var networkInternetCmd = &cobra.Command{
	Use:   "internet",
	Short: "Switch to the internet network profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runNetworkDown(networkRPCConn); err != nil {
			return err
		}
		if err := runNetworkUp(networkInternetConn); err != nil {
			return err
		}
		fmt.Printf("switched to internet network (%s)\n", networkInternetConn)
		return nil
	},
}

var networkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show routing and link status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := printCommandOutput("ip", []string{"route"}); err != nil {
			return err
		}
		fmt.Println()
		return printSpeedStatus(networkIface)
	},
}

func init() {
	networkCmd.PersistentFlags().StringVar(&networkInternetConn, "internet-conn", defaultNetworkInternetConn, "nmcli connection name for internet access")
	networkCmd.PersistentFlags().StringVar(&networkRPCConn, "rpc-conn", defaultNetworkRPCConn, "nmcli connection name for the RPC link")
	networkCmd.PersistentFlags().StringVar(&networkIface, "iface", defaultNetworkIface, "network interface name for status checks")

	networkCmd.AddCommand(networkRpcCmd)
	networkCmd.AddCommand(networkInternetCmd)
	networkCmd.AddCommand(networkStatusCmd)
	rootCmd.AddCommand(networkCmd)
}

func runNetworkDown(conn string) error {
	if strings.TrimSpace(conn) == "" {
		return nil
	}
	_ = runCommand("nmcli", []string{"connection", "down", conn}, io.Discard)
	return nil
}

func runNetworkUp(conn string) error {
	if strings.TrimSpace(conn) == "" {
		return fmt.Errorf("missing network connection name")
	}
	return runCommand("nmcli", []string{"connection", "up", conn}, os.Stderr)
}

func printCommandOutput(bin string, args []string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%s not found in PATH", bin)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommand(bin string, args []string, stderr io.Writer) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%s not found in PATH", bin)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", bin, strings.Join(args, " "), err)
	}
	return nil
}

func printSpeedStatus(iface string) error {
	if strings.TrimSpace(iface) == "" {
		return fmt.Errorf("missing interface name")
	}

	if _, err := exec.LookPath("ethtool"); err != nil {
		return fmt.Errorf("ethtool not found in PATH")
	}
	out, err := exec.Command("ethtool", iface).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ethtool %s: %w", iface, err)
	}

	fmt.Printf("interface: %s\n", iface)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Speed:") {
			fmt.Println(line)
			return nil
		}
	}
	fmt.Println("Speed: unknown")
	return nil
}
