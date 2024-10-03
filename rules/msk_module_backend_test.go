package rules

import (
	"path/filepath"
	"testing"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MskModuleBackend(t *testing.T) {
	rule := NewMskModuleBackendRule()

	tests := []struct {
		Name     string
		Files    map[string]string
		WorkDir  string
		Expected helper.Issues
	}{
		{
			Name:    "backend doesn't have the team's suffix",
			WorkDir: filepath.Join("dev-aws", "msk", "pubsub"),
			Files: map[string]string{"backend.tf": `
terraform {
  backend "s3" {
    bucket = "mybucket"
    key    = "dummy-key"
    region = "us-east-1"
  }
}`},
			Expected: helper.Issues{
				{
					Rule:    rule,
					Message: "backend key must have the team's name 'pubsub' as a suffix. Current value is: dummy-key",
					Range: hcl.Range{
						Filename: "backend.tf",
						Start:    hcl.Pos{Line: 5, Column: 5},
						End:      hcl.Pos{Line: 5, Column: 25},
					},
				},
			},
		},
		{
			Name:  "no terraform config defined",
			Files: map[string]string{"empty.tf": ``},
			Expected: helper.Issues{
				{
					Rule:    rule,
					Message: "an s3 backend should be configured for a kafka MSK module",
					Range:   hcl.Range{},
				},
			},
		},
		{
			Name: "no backend defined",
			Files: map[string]string{"env.tf": `
terraform{
	required_version = ">= 1.5.0"
}`},
			Expected: helper.Issues{
				{
					Rule:    rule,
					Message: "an s3 backend should be configured for a kafka MSK module",
					Range:   hcl.Range{},
				},
			},
		},
		{
			Name: "backend is not s3",
			Files: map[string]string{"backend.tf": `
terraform {
  backend "local" {
  }
}`},
			Expected: helper.Issues{
				{
					Rule:    rule,
					Message: "backend should always be s3 for a kafka MSK module",
					Range: hcl.Range{
						Filename: "backend.tf",
						Start:    hcl.Pos{Line: 3, Column: 3},
						End:      hcl.Pos{Line: 3, Column: 18},
					},
				},
			},
		},
		{
			Name: "backend doesn't specify properties",
			Files: map[string]string{"backend.tf": `
terraform {
  backend "s3" {
  }
}`},
			Expected: helper.Issues{
				{
					Rule:    rule,
					Message: "the s3 backend should specify the details inside the kafka MSK module",
					Range: hcl.Range{
						Filename: "backend.tf",
						Start:    hcl.Pos{Line: 3, Column: 3},
						End:      hcl.Pos{Line: 3, Column: 15},
					},
				},
			},
		},
		{
			Name:    "backend defined in second terraform config",
			WorkDir: filepath.Join("dev-aws", "msk", "pubsub"),
			Files: map[string]string{
				"env.tf": `
terraform{
	required_version = ">= 1.5.0"
}`,
				"backend.tf": `
terraform {
  backend "s3" {
	bucket = "mybucket"
	key    = "good-key-team-pubsub"
	region = "us-east-1"
  }
}`,
			},
			Expected: []*helper.Issue{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			runner := WithWorkDir(helper.TestRunner(t, test.Files), test.WorkDir)

			if err := rule.Check(runner); err != nil {
				require.NoError(t, err, "Unexpected error occurred")
			}

			helper.AssertIssues(t, test.Expected, runner.Issues)
		})
	}
}

type RunnerWithWorkDir struct {
	*helper.Runner
	workDir string
}

// WithWorkDir constructs a runner that always returns the set workdir when calling Originalwd.
func WithWorkDir(h *helper.Runner, workDir string) *RunnerWithWorkDir {
	return &RunnerWithWorkDir{Runner: h, workDir: workDir}
}

// Returns the set workdir.
func (r *RunnerWithWorkDir) GetOriginalwd() (string, error) {
	return r.workDir, nil
}
