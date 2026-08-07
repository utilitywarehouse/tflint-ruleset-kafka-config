package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKTopicMaxMessageBytesRule_not_specified(t *testing.T) {
	rule := &MSKTopicMaxMessageBytesRule{}

	input := `
resource "kafka_topic" "topic_without_max_message_bytes" {
  name = "topic_without_max_message_bytes"
  config = {
    "cleanup.policy"   = "compact"
    "compression.type" = "zstd"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	helper.AssertIssues(t, []*helper.Issue{}, runner.Issues)
	assert.Empty(t, runner.Changes())
}

func Test_MSKTopicMaxMessageBytesRule_within_limit(t *testing.T) {
	rule := &MSKTopicMaxMessageBytesRule{}

	input := `
resource "kafka_topic" "topic_with_max_message_bytes" {
  name = "topic_with_max_message_bytes"
  config = {
    "cleanup.policy"    = "compact"
    "compression.type"  = "zstd"
    "max.message.bytes" = "3145728"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	helper.AssertIssues(t, []*helper.Issue{}, runner.Issues)
	assert.Empty(t, runner.Changes())
}

func Test_MSKTopicMaxMessageBytesRule_exceeds_limit(t *testing.T) {
	rule := &MSKTopicMaxMessageBytesRule{}

	input := `
resource "kafka_topic" "topic_with_large_max_message_bytes" {
  name = "topic_with_large_max_message_bytes"
  config = {
    "cleanup.policy"    = "compact"
    "compression.type"  = "zstd"
    "max.message.bytes" = "3145729"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	expected := []*helper.Issue{
		{
			Message: "max.message.bytes must be less than or equal to 3145728 bytes (3MB)",
			Range: hcl.Range{
				Filename: fileName,
				Start:    hcl.Pos{Line: 7, Column: 27},
				End:      hcl.Pos{Line: 7, Column: 36},
			},
			Rule: rule,
		},
	}

	helper.AssertIssues(t, expected, runner.Issues)
	assert.Empty(t, runner.Changes())
}

func Test_MSKTopicMaxMessageBytesRule_invalid_value(t *testing.T) {
	rule := &MSKTopicMaxMessageBytesRule{}

	input := `
resource "kafka_topic" "topic_with_invalid_max_message_bytes" {
  name = "topic_with_invalid_max_message_bytes"
  config = {
    "cleanup.policy"    = "compact"
    "compression.type"  = "zstd"
    "max.message.bytes" = "invalid-val"
  }
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	expected := []*helper.Issue{
		{
			Message: "max.message.bytes must have a valid integer value expressed in bytes",
			Range: hcl.Range{
				Filename: fileName,
				Start:    hcl.Pos{Line: 7, Column: 27},
				End:      hcl.Pos{Line: 7, Column: 40},
			},
			Rule: rule,
		},
	}

	helper.AssertIssues(t, expected, runner.Issues)
	assert.Empty(t, runner.Changes())
}

func Test_MSKTopicMaxMessageBytesRule_no_config(t *testing.T) {
	rule := &MSKTopicMaxMessageBytesRule{}

	input := `
resource "kafka_topic" "topic_without_config" {
  name = "topic_without_config"
}`

	runner := helper.TestRunner(t, map[string]string{fileName: input})
	require.NoError(t, rule.Check(runner))

	helper.AssertIssues(t, []*helper.Issue{}, runner.Issues)
	assert.Empty(t, runner.Changes())
}
