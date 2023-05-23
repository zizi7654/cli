package copy

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/pkg/cmd/project/shared/format"
	"github.com/cli/cli/v2/pkg/cmd/project/shared/queries"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
)

type copyOpts struct {
	includeDraftIssues bool
	number             int32
	ownerID            string
	projectID          string
	sourceOrgOwner     string
	sourceUserOwner    string
	targetOrgOwner     string
	targetUserOwner    string
	title              string
	format             string
}

type copyConfig struct {
	tp     *tableprinter.TablePrinter
	client *queries.Client
	opts   copyOpts
}

type copyProjectMutation struct {
	CopyProjectV2 struct {
		ProjectV2 queries.Project `graphql:"projectV2"`
	} `graphql:"copyProjectV2(input:$input)"`
}

func NewCmdCopy(f *cmdutil.Factory, runF func(config copyConfig) error) *cobra.Command {
	opts := copyOpts{}
	copyCmd := &cobra.Command{
		Short: "Copy a project",
		Use:   "copy [<number>]",
		Example: heredoc.Doc(`
			# copy project "1" owned by user monalisa to the github org
			gh project copy 1 --source-user monalisa --target-org github --title "a new project"
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.MutuallyExclusive(
				"only one of `--source-user` or `--source-org` may be used",
				opts.sourceUserOwner != "",
				opts.sourceOrgOwner != "",
			); err != nil {
				return err
			}

			if err := cmdutil.MutuallyExclusive(
				"only one of `--target-user` or `--target-org` may be used",
				opts.targetUserOwner != "",
				opts.targetOrgOwner != "",
			); err != nil {
				return err
			}

			client, err := queries.NewClient()
			if err != nil {
				return err
			}

			if len(args) == 1 {
				num, err := strconv.ParseInt(args[0], 10, 32)
				if err != nil {
					return cmdutil.FlagErrorf("invalid number: %v", args[0])
				}
				opts.number = int32(num)
			}

			t := tableprinter.New(f.IOStreams)
			config := copyConfig{
				tp:     t,
				client: client,
				opts:   opts,
			}

			// allow testing of the command without actually running it
			if runF != nil {
				return runF(config)
			}
			return runCopy(config)
		},
	}

	copyCmd.Flags().StringVar(&opts.sourceUserOwner, "source-user", "", "Login of the source user owner. Use \"@me\" for the current user.")
	copyCmd.Flags().StringVar(&opts.sourceOrgOwner, "source-org", "", "Login of the source organization owner")
	copyCmd.Flags().StringVar(&opts.targetUserOwner, "target-user", "", "Login of the target organization owner. Use \"@me\" for the current user.")
	copyCmd.Flags().StringVar(&opts.targetOrgOwner, "target-org", "", "Login of the target organization owner")
	copyCmd.Flags().StringVar(&opts.title, "title", "", "Title for the new project")
	copyCmd.Flags().BoolVar(&opts.includeDraftIssues, "drafts", false, "Include draft issues when copying")
	cmdutil.StringEnumFlag(copyCmd, &opts.format, "format", "", "", []string{"json"}, "Output format")

	_ = copyCmd.MarkFlagRequired("title")

	return copyCmd
}

func runCopy(config copyConfig) error {
	sourceOwner, err := config.client.NewOwner(config.opts.sourceUserOwner, config.opts.sourceOrgOwner)
	if err != nil {
		return err
	}

	targetOwner, err := config.client.NewOwner(config.opts.targetUserOwner, config.opts.targetOrgOwner)
	if err != nil {
		return err
	}

	project, err := config.client.NewProject(sourceOwner, config.opts.number, false)
	if err != nil {
		return err
	}

	config.opts.projectID = project.ID
	config.opts.ownerID = targetOwner.ID

	query, variables := copyArgs(config)

	err = config.client.Mutate("CopyProjectV2", query, variables)
	if err != nil {
		return err
	}

	if config.opts.format == "json" {
		return printJSON(config, query.CopyProjectV2.ProjectV2)
	}

	return printResults(config, query.CopyProjectV2.ProjectV2)
}

func copyArgs(config copyConfig) (*copyProjectMutation, map[string]interface{}) {
	return &copyProjectMutation{}, map[string]interface{}{
		"input": githubv4.CopyProjectV2Input{
			OwnerID:            githubv4.ID(config.opts.ownerID),
			ProjectID:          githubv4.ID(config.opts.projectID),
			Title:              githubv4.String(config.opts.title),
			IncludeDraftIssues: githubv4.NewBoolean(githubv4.Boolean(config.opts.includeDraftIssues)),
		},
		"firstItems":  githubv4.Int(0),
		"afterItems":  (*githubv4.String)(nil),
		"firstFields": githubv4.Int(0),
		"afterFields": (*githubv4.String)(nil),
	}
}

func printResults(config copyConfig, project queries.Project) error {
	// using table printer here for consistency in case it ends up being needed in the future
	config.tp.AddField(fmt.Sprintf("Created project copy '%s'", project.Title))
	config.tp.EndRow()
	config.tp.AddField(project.URL)
	config.tp.EndRow()
	return config.tp.Render()
}

func printJSON(config copyConfig, project queries.Project) error {
	b, err := format.JSONProject(project)
	if err != nil {
		return err
	}
	config.tp.AddField(string(b))
	return config.tp.Render()
}