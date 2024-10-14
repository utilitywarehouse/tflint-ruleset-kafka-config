package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MskTopicConfigRule(t *testing.T) {
	rule := &MskTopicConfigRule{}

	const fileName = "topics.tf"
	for _, tc := range []struct {
		name     string
		input    string
		fixed    string
		expected helper.Issues
	}{
		{
			name: "missing replication factor",
			input: `
resource "kafka_topic" "topic_without_repl_factor" {
  name = "topic_without_repl_factor"
}`,
			fixed: `
resource "kafka_topic" "topic_without_repl_factor" {
  name               = "topic_without_repl_factor"
  replication_factor = 3
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing replication_factor: it must be equal to '3'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 2, Column: 1},
						End:      hcl.Pos{Line: 2, Column: 51},
					},
				},
			},
		},
		{
			name: "incorrect replication factor",
			input: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 10
}`,
			fixed: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 3
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "the replication_factor must be equal to '3'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 4, Column: 3},
						End:      hcl.Pos{Line: 4, Column: 26},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{fileName: tc.input})
			require.NoError(t, rule.Check(runner))
			helper.AssertIssues(t, tc.expected, runner.Issues)

			if tc.fixed != "" {
				helper.AssertChanges(t, map[string]string{fileName: tc.fixed}, runner.Changes())
			}
		})
	}
}
