package webhooks

import (
	"github.com/cli/cli/v2/pkg/cmd/webhooks/forward"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

func NewCmdRun(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhooks <command>",
		Short: "Interact with GitHub Webhooks",
		Long:  "Create and interact with GitHub Webhooks for an easier development experience.",
	}
	cmdutil.EnableRepoOverride(cmd, f)

	cmd.AddCommand(forward.NewCmdForward(f))

	return cmd
}
