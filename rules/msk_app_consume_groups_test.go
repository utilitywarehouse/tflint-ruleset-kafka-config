package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKAppConsumeGroupsRule(t *testing.T) {
	rule := &MSKAppConsumeGroupsRule{}

	for _, tc := range []struct {
		name     string
		files    map[string]string
		expected helper.Issues
	}{
		{
			name: "single bad entry",
			files: map[string]string{
				"file.tf": `
module "my-app" {
	consume_groups = ["my-bad-group"]
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'consume_groups' must be prefixed with the name of the team using it, but 'my-bad-group' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 35},
					},
				},
			},
		},
		{
			name: "multiple bad entres",
			files: map[string]string{
				"file.tf": `
module "my-app" {
	consume_groups = [
		"my-bad-group1",
		"my-bad-group2",
	]
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'consume_groups' must be prefixed with the name of the team using it, but 'my-bad-group1' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 6, Column: 3},
					},
				},
				{
					Rule:    rule,
					Message: "'consume_groups' must be prefixed with the name of the team using it, but 'my-bad-group2' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 6, Column: 3},
					},
				},
			},
		},
		{
			name: "no issue on valid names",
			files: map[string]string{
				"file.tf": `
module "my-app" {
	consume_groups = ["my-team.my-group1, my-team.my-group2"]
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
