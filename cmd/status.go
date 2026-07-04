package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running instance status",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := runtime.NewManager()
		if err != nil {
			return err
		}
		running, err := mgr.List()
		if err != nil {
			return err
		}

		if statusJSON {
			type entry struct {
				ModelKey    string `json:"model_key"`
				ModelName   string `json:"model_name"`
				ProfileKey  string `json:"profile_key"`
				ProfileName string `json:"profile_name"`
				Port        int    `json:"port"`
				PID         int    `json:"pid"`
				Status      string `json:"status"`
				UptimeSec   int64  `json:"uptime_seconds"`
			}
			now := time.Now().Unix()
			entries := make([]entry, len(running))
			for i, r := range running {
				entries[i] = entry{
					ModelKey:    r.ModelKey,
					ModelName:   r.ModelName,
					ProfileKey:  r.ProfileKey,
					ProfileName: r.ProfileName,
					Port:        r.Port,
					PID:         r.PID,
					Status:      string(health.Check(r.Port)),
					UptimeSec:   now - r.StartedAt,
				}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		if len(running) == 0 {
			fmt.Println("nothing running")
			return nil
		}

		now := time.Now().Unix()
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "MODEL\tPROFILE\tPORT\tPID\tSTATUS\tUPTIME")
		for _, r := range running {
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
				r.ModelName, r.ProfileName, r.Port, r.PID,
				health.Check(r.Port), formatUptime(now-r.StartedAt))
		}
		return w.Flush()
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(statusCmd)
}

func formatUptime(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh%dm", seconds/3600, (seconds%3600)/60)
}
