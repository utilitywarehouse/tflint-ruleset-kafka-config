package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

//nolint:maintidx
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
    "retention.ms"     = "86400000"
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
    "retention.ms"     = "86400000"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_repl_factor" {
  name               = "topic_without_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    "retention.ms"     = "86400000"
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
    "retention.ms"     = "86400000"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_incorrect_repl_factor" {
  name               = "topic_with_incorrect_repl_factor"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    "retention.ms"     = "86400000"
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
    "retention.ms"     = "86400000"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_compression_type" {
  name               = "topic_without_compression_type"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
    "cleanup.policy"   = "delete"
    "retention.ms"     = "86400000"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing compression.type: it must be equal to 'zstd'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 8, Column: 4},
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
    "retention.ms"     = "86400000"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_wrong_compression_type" {
  name               = "topic_with_wrong_compression_type"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    "retention.ms"     = "86400000"
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
    "retention.ms"     = "86400000"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_cleanup_policy" {
  name               = "topic_without_cleanup_policy"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
    "retention.ms"     = "86400000"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "missing cleanup.policy: using default 'delete'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 8, Column: 4},
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
			name: "no retention on topic with delete policy",
			input: `
resource "kafka_topic" "topic_without_retention" {
  name               = "topic_without_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_retention" {
  name               = "topic_without_retention"
  replication_factor = 3
  config = {
    "retention.ms"     = "???"
    "cleanup.policy"   = "delete"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "retention.ms must be defined on a topic with cleanup policy delete",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 8, Column: 4},
					},
				},
			},
		},
		{
			// checking that multiple fixes will be inserted correctly as the deletion policy defaults to delete and a retention template should be inserted in this case
			name: "topic without policy and without retention time",
			input: `
resource "kafka_topic" "topic_without_policy_and_retention" {
  name               = "topic_without_policy_and_retention"
  replication_factor = 3
  config = {
    "compression.type" = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_without_policy_and_retention" {
  name               = "topic_without_policy_and_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "???"
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
				{
					Rule:    rule,
					Message: "retention.ms must be defined on a topic with cleanup policy delete",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 7, Column: 4},
					},
				},
			},
		},
		{
			name: "invalid retention time",
			input: `
resource "kafka_topic" "topic_with_invalid_retention" {
  name               = "topic_with_invalid_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "???"
    "compression.type" = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "retention.ms must have a valid integer value expressed in milliseconds. Use -1 for infinite retention",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 7, Column: 26},
						End:      hcl.Pos{Line: 7, Column: 31},
					},
				},
			},
		},
		{
			name: "retention time bigger than 3 days requires tiered storage",
			input: `
resource "kafka_topic" "topic_with_more_than_3_days_retention" {
  name               = "topic_with_more_than_3_days_retention"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "259200001"
    "compression.type" = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_more_than_3_days_retention" {
  name               = "topic_with_more_than_3_days_retention"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "cleanup.policy"     = "delete"
    "retention.ms"       = "259200001"
    "compression.type"   = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "tiered storage should be enabled when retention time is longer than 3 days",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 9, Column: 4},
					},
				},
				{
					Rule:    rule,
					Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 9, Column: 4},
					},
				},
			},
		},
		{
			name: "forgot tiered storage enabling",
			input: `
resource "kafka_topic" "topic_with_missing_tiered_storage_enabling" {
  name               = "topic_with_missing_tiered_storage_enabling"
  replication_factor = 3
  config = {
    "cleanup.policy"   = "delete"
    "retention.ms"     = "259200001"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "compression.type" = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_missing_tiered_storage_enabling" {
  name               = "topic_with_missing_tiered_storage_enabling"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    "retention.ms"          = "259200001"
    # keep data in hot storage for 1 day
    "local.retention.ms" = "86400000"
    "compression.type"   = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "tiered storage should be enabled when retention time is longer than 3 days",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 11, Column: 4},
					},
				},
			},
		},
		{
			name: "tiered storage disabled for retention period bigger than 3 days",
			input: `
resource "kafka_topic" "topic_with_more_than_3_days_retention_tiered_disabled" {
  name               = "topic_with_more_than_3_days_retention_tiered_disabled"
  replication_factor = 3
  config = {
    "remote.storage.enable" = "false"
    "cleanup.policy"        = "delete"
    "retention.ms"          = "259200001"
    "compression.type"      = "zstd"
  }
}`,
			fixed: `
resource "kafka_topic" "topic_with_more_than_3_days_retention_tiered_disabled" {
  name               = "topic_with_more_than_3_days_retention_tiered_disabled"
  replication_factor = 3
  config = {
    # keep data in hot storage for 1 day
    "local.retention.ms"    = "86400000"
    "remote.storage.enable" = "true"
    "cleanup.policy"        = "delete"
    "retention.ms"          = "259200001"
    "compression.type"      = "zstd"
  }
}`,
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "tiered storage should be enabled when retention time is longer than 3 days",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 6, Column: 31},
						End:      hcl.Pos{Line: 6, Column: 38},
					},
				},
				{
					Rule:    rule,
					Message: "missing local.retention.ms when tiered storage is enabled: using default '86400000'",
					Range: hcl.Range{
						Filename: fileName,
						Start:    hcl.Pos{Line: 5, Column: 3},
						End:      hcl.Pos{Line: 10, Column: 4},
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
    "retention.ms"     = "86400000"
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
