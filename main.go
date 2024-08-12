// Generate Github Actions configurations from Dagger pipelines
//
// Daggerizing your CI makes your YAML configurations smaller, but they still exist,
// and they're still a pain to maintain by hand.
//
// This module aims to finish the job, by letting you generate your remaining
// YAML configuration from a Dagger pipeline, written in your favorite language.
package main

import (
	"dagger/dagger-2-gha/internal/dagger"
	"fmt"
	"strings"
)

func New(
	// Disable sending traces to Dagger Cloud
	// +optional
	noTraces bool,
	// Public Dagger Cloud token, for open-source projects. DO NOT PASS YOUR PRIVATE DAGGER CLOUD TOKEN!
	// This is for a special "public" token which can safely be shared publicly.
	// To get one, contact support@dagger.io
	// +optional
	publicToken string,
	// Dagger version to run in the Github Actions pipelines
	// +optional
	// +default="latest"
	daggerVersion string,
) *Dagger2Gha {
	return &Dagger2Gha{
		PublicToken:   publicToken,
		NoTraces:      noTraces,
		DaggerVersion: daggerVersion,
	}
}

type Dagger2Gha struct {
	// +private
	PushTriggers []PushTrigger
	// +private
	PullRequestTriggers []PullRequestTrigger
	// +private
	PublicToken string
	// +private
	DaggerVersion string
	// +private
	NoTraces bool
}

// Add a trigger to execute a Dagger pipeline on a git push
func (m *Dagger2Gha) OnPush(
	// The Dagger command to execute
	// Example 'build --source=.'
	command string,
	// +optional
	// +default="."
	module string,
	// +optional
	branches []string,
	// +optional
	tags []string,
) *Dagger2Gha {
	m.PushTriggers = append(m.PushTriggers, PushTrigger{
		Event: PushEvent{
			Branches: branches,
			Tags:     tags,
		},
		Pipeline: m.pipeline(command, module),
	})
	return m
}

// Add a trigger to execute a Dagger pipeline on a pull request
func (m *Dagger2Gha) OnPullRequest(
	// The Dagger command to execute
	// Example 'build --source=.'
	command string,
	// +optional
	// +default="."
	module string,
	// +optional
	branches []string,
) *Dagger2Gha {
	m.PullRequestTriggers = append(m.PullRequestTriggers, PullRequestTrigger{
		Event: PullRequestEvent{
			Branches: branches,
		},
		Pipeline: m.pipeline(command, module),
	})
	return m
}

func (m *Dagger2Gha) pipeline(
	// The Dagger command to execute
	// Example 'build --source=.'
	command string,
	// +optional
	// +default="."
	module string,
) Pipeline {
	return Pipeline{
		DaggerVersion: m.DaggerVersion,
		PublicToken:   m.PublicToken,
		NoTraces:      m.NoTraces,
		Command:       command,
		Module:        module,
	}
}

// Generate a github config directory, usable as an overlay on the repository root
func (m *Dagger2Gha) Config() *dagger.Directory {
	dir := dag.Directory()
	for i, t := range m.PushTriggers {
		filename := fmt.Sprintf("push-%d.yml", i+1)
		dir = dir.WithDirectory(".", t.Config(filename))
	}
	for i, t := range m.PullRequestTriggers {
		filename := fmt.Sprintf("pr-%d.yml", i+1)
		dir = dir.WithDirectory(".", t.Config(filename))
	}
	return dir
}

type PushTrigger struct {
	Event    PushEvent
	Pipeline Pipeline
}

func (t PushTrigger) asWorkflow() Workflow {
	var workflow = t.Pipeline.asWorkflow()
	workflow.On = WorkflowTriggers{Push: &(t.Event)}
	return workflow
}

func (t PushTrigger) Config(filename string) *dagger.Directory {
	return t.asWorkflow().Config(filename)
}

type PullRequestTrigger struct {
	Event    PullRequestEvent
	Pipeline Pipeline
}

func (t PullRequestTrigger) asWorkflow() Workflow {
	var workflow = t.Pipeline.asWorkflow()
	workflow.On = WorkflowTriggers{PullRequest: &(t.Event)}
	return workflow
}

func (t PullRequestTrigger) Config(filename string) *dagger.Directory {
	return t.asWorkflow().Config(filename)
}

type Pipeline struct {
	// +private
	DaggerVersion string
	// +private
	PublicToken string
	// +private
	Module string
	// +private
	Command string
	// +private
	NoTraces bool
}

func (p *Pipeline) Name() string {
	return strings.SplitN(p.Command, " ", 2)[0]
}

// Generate a GHA workflow from a Dagger pipeline definition.
// The workflow will have no triggers, they should be filled separately.
func (p *Pipeline) asWorkflow() Workflow {
	return Workflow{
		Name: p.Command,
		On:   WorkflowTriggers{}, // Triggers intentionally left blank
		Jobs: map[string]Job{
			"dagger": Job{
				RunsOn: "ubuntu-latest",
				Steps: []JobStep{
					p.checkoutStep(),
					p.callDaggerStep(),
				},
			},
		},
	}
}

func (p *Pipeline) checkoutStep() JobStep {
	return JobStep{
		Name: "Checkout",
		Uses: "actions/checkout@v4",
	}
}

func (p *Pipeline) callDaggerStep() JobStep {
	step := JobStep{
		Name: "Call Dagger",
		Uses: "dagger/dagger-for-github@v6",
		With: map[string]string{
			"version": "latest",
			"module":  p.Module,
			"args":    p.Command,
		},
	}
	if !p.NoTraces {
		if p.PublicToken != "" {
			step.With["cloud-token"] = p.PublicToken
		} else {
			step.With["cloud-token"] = "${{ secrets.DAGGER_CLOUD_TOKEN }}"
		}
	}
	return step
}

func (p *Pipeline) githubAction() Action {
	var env = make(map[string]string)
	if p.PublicToken != "" {
		env["DAGGER_CLOUD_TOKEN"] = p.PublicToken
	}
	action := Action{
		Name: p.Name(),
		Runs: Runs{
			Using: "composite",
			Steps: []CompositeActionStep{
				CompositeActionStep{
					Name: "Checkout",
					Uses: "actions/checkout@v4",
				},
				CompositeActionStep{
					Name: "Dagger",
					Uses: "dagger/dagger-for-github@v6",
					With: map[string]string{
						"version": p.DaggerVersion,
						"command": p.Command,
						"module":  p.Module,
					},
					Env: env,
				},
			},
		},
	}

	return action
}
