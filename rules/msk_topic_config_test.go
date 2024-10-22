package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKTopicConfigRule(t *testing.T) {
	rule := &MSKTopicConfigRule{}

	const fileName = "topics.tf"
	for _, tc := range []struct {
		name     string
		input    string
		fixed    string
		expected helper.Issues
	}{
		{
			name: "missing replication factor and topic name not defined",
			input: `
resource "kafka_topic" "topic_without_repl_factor_and_name" {
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing replication_factor: it must be equal to '3'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 2, Column: 1},
						End:      hcl.Pos{Line: 2, Column: 60},
					},
				},
			},
		},

		{
			name: "missing replication factor",
			input: `
resource "kafka_topic" "topic_without_repl_factor" {
  name = "topic_without_repl_factor"
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_repl_factor" {
  name               = "topic_without_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
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
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
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
		{
			name: "missing config attribute",
			input: `
resource "kafka_topic" "topic_without_config" {
  name = "topic_without_config"
  replication_factor = 3
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing config attribute: the topic configuration must be specified in a config attribute",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 2, Column: 1},
						End:      hcl.Pos{Line: 2, Column: 46},
					},
				},
			},
		},
		{
			name: "missing compression type",
			input: `
resource "kafka_topic" "topic_without_compression_type" {
  name               = "topic_without_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_compression_type" {
  name               = "topic_without_compression_type"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing compression.type: it must be equal to 'zstd'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 7, Column: 4},
					},
				},
			},
		},
		{
			name: "wrong compression type",
			input: `
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name               = "topic_with_wrong_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "gzip"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name               = "topic_with_wrong_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "the compression.type value must be equal to 'zstd'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 7, Column: 26},
						End:      hcl.Pos{Line: 7, Column: 32},
					},
				},
			},
		},
		{
			name: "missing cleanup policy",
			input: `
resource "kafka_topic" "topic_without_cleanup_policy" {
  name               = "topic_without_cleanup_policy"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_cleanup_policy" {
  name               = "topic_without_cleanup_policy"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing cleanup.policy: using default 'delete'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 7, Column: 4},
					},
				},
			},
		},
		{
			name: "invalid cleanup policy value",
			input: `
resource "kafka_topic" "topic_with_invalid_cleanup_policy" {
  name               = "topic_with_invalid_cleanup_policy"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "invalid-value"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "invalid cleanup.policy: it must be one of [delete, compact], but currently is 'invalid-value'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 6, Column: 26},
						End:      hcl.Pos{Line: 6, Column: 41},
					},
				},
			},
		},
		{
			name: "good topic definition",
			input: `
resource "kafka_topic" "good topic" {
  name               = "good_topic"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{fileName: tc.input})
			require.NoError(t, rule.Check(runner))
			helper.AssertIssues(t, tc.expected, runner.Issues)

			if tc.fixed != "" {
				t.Logf("Proposed changes: %s", string(runner.Changes()[fileName]))
				helper.AssertChanges(t, map[string]string{fileName: tc.fixed}, runner.Changes())
			} else {
				assert.Empty(t, runner.Changes())
			}
		})
	}
}
