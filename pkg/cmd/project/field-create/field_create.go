package fieldcreate

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

type createFieldOpts struct {
	name                string
	dataType            string
	userOwner           string
	singleSelectOptions []string
	orgOwner            string
	number              int32
	projectID           string
	format              string
}

type createFieldConfig struct {
	tp     *tableprinter.TablePrinter
	client *queries.Client
	opts   createFieldOpts
}

type createProjectV2FieldMutation struct {
	CreateProjectV2Field struct {
		Field queries.ProjectField `graphql:"projectV2Field"`
	} `graphql:"createProjectV2Field(input:$input)"`
}

func NewCmdCreateField(f *cmdutil.Factory, runF func(config createFieldConfig) error) *cobra.Command {
	opts := createFieldOpts{}
	createFieldCmd := &cobra.Command{
		Short: "Create a field in a project",
		Use:   "field-create [<number>]",
		Example: heredoc.Doc(`
			# create a field in the current user's project "1"
			gh project field-create 1 --user "@me" --name "new field" --data-type "text"

			# create a field with three options to select from
			gh project field-create 1 --user monalisa --name "new field" --data-type "SINGLE_SELECT" --single-select-options "one,two,three"
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.MutuallyExclusive(
				"only one of `--user` or `--org` may be used",
				opts.userOwner != "",
				opts.orgOwner != "",
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
			config := createFieldConfig{
				tp:     t,
				client: client,
				opts:   opts,
			}

			if config.opts.dataType == "SINGLE_SELECT" && len(config.opts.singleSelectOptions) == 0 {
				return fmt.Errorf("passing `--single-select-options` is required for SINGLE_SELECT data type")
			}

			// allow testing of the command without actually running it
			if runF != nil {
				return runF(config)
			}
			return runCreateField(config)
		},
	}

	createFieldCmd.Flags().StringVar(&opts.userOwner, "user", "", "Login of the user owner. Use \"@me\" for the current user.")
	createFieldCmd.Flags().StringVar(&opts.orgOwner, "org", "", "Login of the organization owner")
	createFieldCmd.Flags().StringVar(&opts.name, "name", "", "Name of the new field")
	cmdutil.StringEnumFlag(createFieldCmd, &opts.dataType, "data-type", "", "", []string{"TEXT", "SINGLE_SELECT", "DATE", "NUMBER"}, "DataType of the new field.")
	createFieldCmd.Flags().StringSliceVar(&opts.singleSelectOptions, "single-select-options", []string{}, "Options for SINGLE_SELECT data type")
	cmdutil.StringEnumFlag(createFieldCmd, &opts.format, "format", "", "", []string{"json"}, "Output format")

	_ = createFieldCmd.MarkFlagRequired("name")
	_ = createFieldCmd.MarkFlagRequired("data-type")

	return createFieldCmd
}

func runCreateField(config createFieldConfig) error {
	owner, err := config.client.NewOwner(config.opts.userOwner, config.opts.orgOwner)
	if err != nil {
		return err
	}

	project, err := config.client.NewProject(owner, config.opts.number, false)
	if err != nil {
		return err
	}
	config.opts.projectID = project.ID

	query, variables := createFieldArgs(config)

	err = config.client.Mutate("CreateField", query, variables)
	if err != nil {
		return err
	}

	if config.opts.format == "json" {
		return printJSON(config, query.CreateProjectV2Field.Field)
	}

	return printResults(config, query.CreateProjectV2Field.Field)
}

func createFieldArgs(config createFieldConfig) (*createProjectV2FieldMutation, map[string]interface{}) {
	input := githubv4.CreateProjectV2FieldInput{
		ProjectID: githubv4.ID(config.opts.projectID),
		DataType:  githubv4.ProjectV2CustomFieldType(config.opts.dataType),
		Name:      githubv4.String(config.opts.name),
	}

	if len(config.opts.singleSelectOptions) != 0 {
		opts := make([]githubv4.ProjectV2SingleSelectFieldOptionInput, 0)
		for _, opt := range config.opts.singleSelectOptions {
			opts = append(opts, githubv4.ProjectV2SingleSelectFieldOptionInput{
				Name:  githubv4.String(opt),
				Color: githubv4.ProjectV2SingleSelectFieldOptionColor("GRAY"),
			})
		}
		input.SingleSelectOptions = &opts
	}

	return &createProjectV2FieldMutation{}, map[string]interface{}{
		"input": input,
	}
}

func printResults(config createFieldConfig, field queries.ProjectField) error {
	// using table printer here for consistency in case it ends up being needed in the future
	config.tp.AddField("Created field")
	config.tp.EndRow()
	return config.tp.Render()
}

func printJSON(config createFieldConfig, field queries.ProjectField) error {
	b, err := format.JSONProjectField(field)
	if err != nil {
		return err
	}
	config.tp.AddField(string(b))
	return config.tp.Render()
}