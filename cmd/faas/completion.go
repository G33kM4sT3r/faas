package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{ //nolint:gochecknoglobals // cobra command
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for faas.

To load completions in the current bash session:
  source <(faas completion bash)

To install permanently:
  faas completion bash > /etc/bash_completion.d/faas       # Linux
  faas completion bash > $(brew --prefix)/etc/bash_completion.d/faas  # macOS
  faas completion zsh  > "${fpath[1]}/_faas"
  faas completion fish > ~/.config/fish/completions/faas.fish`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}
