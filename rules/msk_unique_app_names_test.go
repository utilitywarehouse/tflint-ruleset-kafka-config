package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKUniqueAppNamesRule(t *testing.T) {
	rule := &MSKUniqueAppNamesRule{}

	for _, tc := range []struct {
		name     string
		files    map[string]string
		expected helper.Issues
	}{
		{
			name: "reports duplicate app names in same file",
			files: map[string]string{
				"file.tf": `
module "first_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}

module "second_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'cert_common_name' must be unique across a module, but 'my-namespace/my-app' has already been seen",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 9, Column: 3},
						End:      hcl.Pos{Line: 9, Column: 43},
					},
				},
			},
		},
		{
			name: "reports duplicate app names across files",
			files: map[string]string{
				"first.tf": `
module "first_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}
`,
				"second.tf": `
module "second_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'cert_common_name' must be unique across a module, but 'my-namespace/my-app' has already been seen",
					Range: hcl.Range{
						Filename: "second.tf",
						Start:    hcl.Pos{Line: 4, Column: 3},
						End:      hcl.Pos{Line: 4, Column: 43},
					},
				},
			},
		},
		{
			name: "reports repeated duplicate app names",
			files: map[string]string{
				"file.tf": `
module "first_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}

module "second_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}

module "third_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/my-app"
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'cert_common_name' must be unique across a module, but 'my-namespace/my-app' has already been seen",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 9, Column: 3},
						End:      hcl.Pos{Line: 9, Column: 43},
					},
				},
				{
					Rule:    rule,
					Message: "'cert_common_name' must be unique across a module, but 'my-namespace/my-app' has already been seen",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 14, Column: 3},
						End:      hcl.Pos{Line: 14, Column: 43},
					},
				},
			},
		},
		{
			name: "Reports nothing with all unique names",
			files: map[string]string{
				"file.tf": `
module "first_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/first-app"
}

module "second_app" {
  source           = "../../../modules/tls-app"
  cert_common_name = "my-namespace/second-app"
}
`,
			},
			expected: []*helper.Issue{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := helper.TestRunner(t, tc.files)

			require.NoError(t, rule.Check(runner))

			helper.AssertIssues(t, tc.expected, runner.Issues)
		})
	}
}
