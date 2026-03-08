package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/G33kM4sT3r/faas/internal/runtime"
	"github.com/G33kM4sT3r/faas/internal/state"
	"github.com/G33kM4sT3r/faas/internal/ui"
)

var ( //nolint:gochecknoglobals // cobra flag variables
	lsJSON  bool
	lsQuiet bool
)

var lsCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List deployed functions",
	RunE:    runLs,
}

func setupLsFlags() {
	lsCmd.Flags().BoolVar(&lsJSON, "json", false, "output as JSON")
	lsCmd.Flags().BoolVarP(&lsQuiet, "quiet", "q", false, "only print function names")
}

func runLs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fns, err := store.List()
	if err != nil {
		return err
	}

	docker, _ := runtime.NewDocker(ctx)
	if docker != nil {
		for i := range fns {
			status, err := docker.Status(ctx, fns[i].ContainerID)
			if err != nil || !status.Running {
				fns[i].Status = state.StatusStopped
				_ = store.UpdateStatus(fns[i].Name, state.StatusStopped)
			}
		}
	}

	if len(fns) == 0 {
		fmt.Println(ui.StyleDim.Render("No functions deployed"))
		return nil
	}

	if lsJSON {
		data, _ := json.MarshalIndent(fns, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if lsQuiet {
		for i := range fns {
			fmt.Println(fns[i].Name)
		}
		return nil
	}

	fmt.Printf("%-12s %-12s %-8s %-12s %s\n",
		ui.StyleBold.Render("NAME"),
		ui.StyleBold.Render("LANGUAGE"),
		ui.StyleBold.Render("PORT"),
		ui.StyleBold.Render("STATUS"),
		ui.StyleBold.Render("CREATED"),
	)

	for i := range fns {
		statusStr := formatStatus(fns[i].Status)
		age := formatAge(fns[i].CreatedAt)
		fmt.Printf("%-12s %-12s %-8d %-12s %s\n",
			fns[i].Name, fns[i].Language, fns[i].Port, statusStr, ui.StyleDim.Render(age))
	}

	return nil
}

func formatStatus(s state.Status) string {
	switch s {
	case state.StatusHealthy:
		return ui.StyleSuccess.Render(string(s))
	case state.StatusError, state.StatusUnhealthy:
		return ui.StyleError.Render(string(s))
	case state.StatusStopped:
		return ui.StyleDim.Render(string(s))
	default:
		return ui.StyleWarning.Render(string(s))
	}
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
