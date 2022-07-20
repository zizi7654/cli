package cmdutil

import (
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/cli/cli/v2/internal/ghinstance"
)

// NB this could be embedded but having to write out a PromptOpts literal was highly tedious
type PromptOpts struct {
	Message    string
	Validators []Validator
	Help       string
	Default    string
}

type SelectOpts struct {
	Message    string
	Validators []Validator
	Help       string
	Default    string

	Options       []string
	PageSize      int
	VimMode       bool
	FilterMessage string
	Filter        func(filter string, value string, index int) bool
	Description   func(value string, index int) string
}

type ConfirmOpts struct {
	Message    string
	Validators []Validator
	Help       string

	Default bool
}

type Validator func(string) error

func NewPrompter(stdin io.Reader, stdout, stderr io.Writer) Prompter {
	return &surveyPrompter{
		stdin:  stdin.(terminal.FileReader),
		stdout: stdout.(terminal.FileWriter),
		stderr: stderr,
	}
}

type surveyPrompter struct {
	stdin  terminal.FileReader
	stdout terminal.FileWriter
	stderr io.Writer
}

func toAskOpts(vs []Validator) []survey.AskOpt {
	ao := []survey.AskOpt{}
	for _, v := range vs {
		ao = append(ao, survey.WithValidator(func(i interface{}) error {
			return v(i.(string))
		}))
	}
	return ao
}

func wrapSurveyError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("could not prompt: %w", err)
}

func (p *surveyPrompter) Select(opts SelectOpts) (result int, err error) {
	q := &survey.Select{
		Message:       opts.Message,
		Default:       opts.Default,
		Help:          opts.Help,
		Options:       opts.Options,
		PageSize:      opts.PageSize,
		VimMode:       opts.VimMode,
		FilterMessage: opts.FilterMessage,
		Filter:        opts.Filter,
		Description:   opts.Description,
	}

	ao := toAskOpts(opts.Validators)
	ao = append(ao, survey.WithStdio(p.stdin, p.stdout, p.stderr))

	err = wrapSurveyError(survey.AskOne(q, &result, ao...))

	return
}

func (p *surveyPrompter) MultiSelect(opts SelectOpts) (result int, err error) {
	q := &survey.MultiSelect{
		Message:       opts.Message,
		Default:       opts.Default,
		Help:          opts.Help,
		Options:       opts.Options,
		PageSize:      opts.PageSize,
		VimMode:       opts.VimMode,
		FilterMessage: opts.FilterMessage,
		Filter:        opts.Filter,
	}

	ao := toAskOpts(opts.Validators)
	ao = append(ao, survey.WithStdio(p.stdin, p.stdout, p.stderr))

	err = wrapSurveyError(survey.AskOne(q, &result, ao...))

	return
}

func (p *surveyPrompter) ask(q survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	opts = append(opts, survey.WithStdio(p.stdin, p.stdout, p.stderr))
	err := survey.AskOne(q, response, opts...)
	if err == nil {
		return nil
	}
	return fmt.Errorf("could not prompt: %w", err)
}

func (p *surveyPrompter) Input(prompt, defaultValue string) (result string, err error) {
	q := &survey.Input{
		Message: prompt,
		Default: defaultValue,
	}

	err = p.ask(q, &result)

	return
}

func (p *surveyPrompter) InputHostname() (result string, err error) {
	q := &survey.Input{
		Message: "GHE hostname:",
	}

	err = p.ask(q, &result, survey.WithValidator(func(v interface{}) error {
		return ghinstance.HostnameValidator(v.(string))
	}))

	return
}

func (p *surveyPrompter) Password(opts PromptOpts) (result string, err error) {
	q := &survey.Password{
		Message: opts.Message,
		Help:    opts.Help,
	}

	ao := toAskOpts(opts.Validators)
	ao = append(ao, survey.WithStdio(p.stdin, p.stdout, p.stderr))

	err = wrapSurveyError(survey.AskOne(q, &result, ao...))

	return
}

func (p *surveyPrompter) Confirm(opts ConfirmOpts) (result bool, err error) {
	q := &survey.Confirm{
		Message: opts.Message,
		Help:    opts.Help,
		Default: opts.Default,
	}

	ao := toAskOpts(opts.Validators)
	ao = append(ao, survey.WithStdio(p.stdin, p.stdout, p.stderr))

	err = wrapSurveyError(survey.AskOne(q, &result, ao...))

	return
}
