package forward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
)

type ForwardOptions struct {
	HttpClient func() (*http.Client, error)
	Config     func() (config.Config, error)

	Repo   string
	Events []string
	Port   int
}

func NewCmdForward(f *cmdutil.Factory) *cobra.Command {
	opts := &ForwardOptions{
		HttpClient: f.HttpClient,
		Config:     f.Config,
	}

	cmd := &cobra.Command{
		Use:   "forward",
		Short: "Forward webhook events to your localhost server",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return forwardWebhook(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "r", "", "The repository name with owner: user/repo")
	cmd.Flags().IntVarP(&opts.Port, "port", "P", 0, "Port number")
	cmd.Flags().StringSliceVarP(&opts.Events, "events", "E", nil, "The list of events")

	return cmd
}

type body struct {
	Name   string     `json:"name"`
	Active bool       `json:"active"`
	Events []string   `json:"events"`
	Config configBody `json:"config"`
}

type configBody struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	InsecureSsl string `json:"insecure_ssl"`
}

func forwardWebhook(opts *ForwardOptions) error {
	c, err := opts.HttpClient()
	if err != nil {
		return fmt.Errorf("failed to create http client: %w", err)
	}
	client := api.NewClientFromHTTP(c)
	cfg, err := opts.Config()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	host, _ := cfg.DefaultHost()
	if true {
		host = "github.localhost"
	}

	if len(opts.Events) == 0 {
		return fmt.Errorf("missing events")
	}
	if opts.Repo == "" {
		return fmt.Errorf("missing repo")
	}
	if opts.Port <= 0 {
		return fmt.Errorf("missing or invalid port")
	}

	b := body{
		Name:   "dev",
		Active: true,
		Events: opts.Events,
		Config: configBody{
			ContentType: "json",
			InsecureSsl: "0",
		},
	}
	bts, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	p := "http://api.github.localhost/repos/" + opts.Repo + "/hooks"
	if false {
		p = "repos/" + opts.Repo + "/hooks"
	}

	err = client.REST(host, "POST", p, bytes.NewReader(bts), nil)
	if err != nil {
		return fmt.Errorf("failed to create hook: %w", err)
	}

	return nil
}
