package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKTopicNoInfiniteRetentionRule_with_infinite(t *testing.T) {
	rule := &MSKTopicNoInfiniteRetentionRule{}

	input := `
resource "kafka_topic" "topic_with_infinite_retention" {
  name               = "topic_with_infinite_retention"
  config = {
    "retention.ms"          = "-1"
    "local.retention.ms"    = "86400000"
    "cleanup.policy"        = "delete"
    "compression.type"      = "zstd"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	expected := []*helper.Issue{
		{
			Message: infiniteRetentionMsg,
			Range: hcl.Range{
				Filename: fileName,
				Start:    hcl.Pos{Line: 5, Column: 31},
				End:      hcl.Pos{Line: 5, Column: 35},
			},
			Rule: rule,
		},
	}

	helper.AssertIssues(t, expected, runner.Issues)
	assert.Empty(t, runner.Changes())
}

func Test_MSKTopicNoInfiniteRetentionRule_without_infinite(t *testing.T) {
	rule := &MSKTopicNoInfiniteRetentionRule{}

	input := `
resource "kafka_topic" "topic_with_infinite_retention" {
  name               = "topic_with_infinite_retention"
  config = {
    "retention.ms"          = "259200000"
    "local.retention.ms"    = "86400000"
    "cleanup.policy"        = "delete"
    "compression.type"      = "zstd"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	helper.AssertIssues(t, []*helper.Issue{}, runner.Issues)
	assert.Empty(t, runner.Changes())
}
