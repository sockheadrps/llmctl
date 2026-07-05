package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

// psCmd lists currently running model instances.
var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running model instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := runtime.NewManager()
		if err != nil {
			return err
		}

		running, err := mgr.List()
		if err != nil {
			return err
		}

		if len(running) == 0 {
			fmt.Println("nothing running")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "MODEL\tPROFILE\tPORT\tPID\tSTATUS")
		for _, r := range running {
			fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n", r.ModelName, r.ProfileName, r.Port, r.PID, health.Check(r.Host, r.Port))
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
