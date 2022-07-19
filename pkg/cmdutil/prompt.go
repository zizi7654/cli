package cmdutil

import (
	"io"

	"github.com/AlecAivazis/survey/v2"
)

type PromptOpts struct {
	Message    string
	Validators []Validator
	Help       string
	Default    string
}

type SelectOpts struct {
	PromptOpts
	Options       []string
	PageSize      int
	VimMode       bool
	FilterMessage string
	Filter        func(filter string, value string, index int) bool
	Description   func(value string, index int) string
}

type ConfirmOpts struct {
	PromptOpts
	Default bool
}

type Validator func(string) error

func NewPrompter(stdout, stderr io.Writer) Prompter {
	return &surveyPrompter{
		stdout: stdout,
		stderr: stderr,
	}
}

type surveyPrompter struct {
	stdout io.Writer
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

// TODO figure out how to make Survey respect stdout/stderr writers that we set

func (p *surveyPrompter) Select(opts SelectOpts) (string, error) {
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

	var result string

	err := survey.AskOne(q, &result, toAskOpts(opts.Validators)...)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (p *surveyPrompter) MultiSelect(opts SelectOpts) (result string, err error) {
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

	err = survey.AskOne(q, &result, toAskOpts(opts.Validators)...)

	return
}

func (p *surveyPrompter) Input(opts PromptOpts) (result string, err error) {
	q := &survey.Input{
		Message: opts.Message,
		Default: opts.Default,
		Help:    opts.Help,
	}

	err = survey.AskOne(q, &result, toAskOpts(opts.Validators)...)

	return
}

func (p *surveyPrompter) Password(opts PromptOpts) (result string, err error) {
	q := &survey.Password{
		Message: opts.Message,
		Help:    opts.Help,
	}

	err = survey.AskOne(q, &result, toAskOpts(opts.Validators)...)

	return
}

func (p *surveyPrompter) Confirm(opts ConfirmOpts) (result bool, err error) {
	q := &survey.Confirm{
		Message: opts.Message,
		Help:    opts.Help,
		Default: opts.Default,
	}

	err = survey.AskOne(q, &result, toAskOpts(opts.Validators)...)

	return
}
