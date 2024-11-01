package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

var configValueCommentsTests = []topicConfigTestCase{
	{
		name: "retention time without comment",
		input: `
resource "kafka_topic" "topic_without_retention_comment" {
  name = "topic_without_retention_comment"
  config = {
    "retention.ms" = "86400000"
  }
}`, fixed: `
resource "kafka_topic" "topic_without_retention_comment" {
  name = "topic_without_retention_comment"
  config = {
    "retention.ms" = "86400000" # keep data for 1 day
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms must have a comment with the human readable value: adding it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 5},
					End:      hcl.Pos{Line: 5, Column: 19},
				},
			},
		},
	},
	{
		name: "retention time with wrong comment",
		input: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    # keep data for 1 day
    "retention.ms" = "172800000"
  }
}`, fixed: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    # keep data for 2 days
    "retention.ms" = "172800000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "retention.ms value doesn't correspond to the human readable value in the comment: fixing it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 5},
					End:      hcl.Pos{Line: 7, Column: 1},
				},
			},
		},
	},
	{
		name: "retention time good infinite comment",
		input: `
resource "kafka_topic" "topic_good_retention_comment_infinite" {
  name               = "topic_good_retention_comment_infinite"
  replication_factor = 3
  config = {
    # keep data forever
    "retention.ms" = "-1"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "retention time in months",
		input: `
resource "kafka_topic" "topic_good_retention_comment_months" {
  name               = "topic_good_retention_comment_months"
  replication_factor = 3
  config = {
    "retention.ms" = "5184000000" # keep data for 2 months 
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		// the value is validated in the msk_topic_config rule
		name: "retention time invalid",
		input: `
resource "kafka_topic" "topic_def" {
  name               = "topic_def"
  replication_factor = 3
  config = {
    "retention.ms" = "invalid-val"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "retention time in years",
		input: `
resource "kafka_topic" "topic_good_retention_comment_years" {
  name               = "topic_good_retention_comment_years"
  replication_factor = 3
  config = {
    "retention.ms" = "31536000000" # keep data for 1 year 
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "retention time less than a day with good comment",
		input: `
resource "kafka_topic" "topic_good_retention_comment_less_than_a_day" {
  name               = "topic_good_retention_comment_less_than_a_day"
  replication_factor = 3
  config = {
    "retention.ms" = "21600000" # keep data for 6 hours
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "local retention time without comment",
		input: `
resource "kafka_topic" "topic_without_retention_comment" {
  name = "topic_without_retention_comment"
  config = {
    "local.retention.ms" = "86400000"
  }
}`, fixed: `
resource "kafka_topic" "topic_without_retention_comment" {
  name = "topic_without_retention_comment"
  config = {
    "local.retention.ms" = "86400000" # keep data in primary storage for 1 day
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "local.retention.ms must have a comment with the human readable value: adding it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 5, Column: 5},
					End:      hcl.Pos{Line: 5, Column: 25},
				},
			},
		},
	},
	{
		name: "local retention time with wrong comment",
		input: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    # keep data in primary storage for 1 day
    "local.retention.ms" = "3600000"
  }
}`, fixed: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    # keep data in primary storage for 1 hour
    "local.retention.ms" = "3600000"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "local.retention.ms value doesn't correspond to the human readable value in the comment: fixing it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 5},
					End:      hcl.Pos{Line: 7, Column: 1},
				},
			},
		},
	},
	{
		// the value is validated in the msk_topic_config rule
		name: "local retention time invalid",
		input: `
resource "kafka_topic" "topic_def" {
  name               = "topic_def"
  replication_factor = 3
  config = {
    "local.retention.ms" = "invalid-val"
  }
}`,
		expected: []*helper.Issue{},
	},
	{
		name: "max compaction lag without comment",
		input: `
resource "kafka_topic" "topic_def" {
  name               = "topic_def"
  replication_factor = 3
  config = {
    "max.compaction.lag.ms" = "2592000000"
  }
}`, fixed: `
resource "kafka_topic" "topic_def" {
  name               = "topic_def"
  replication_factor = 3
  config = {
    "max.compaction.lag.ms" = "2592000000" # allow not compacted keys maximum for 1 month
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "max.compaction.lag.ms must have a comment with the human readable value: adding it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 5},
					End:      hcl.Pos{Line: 6, Column: 28},
				},
			},
		},
	},
	{
		name: "max compaction lag with wrong comment",
		input: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    "max.compaction.lag.ms" = "3600000" # allow not compacted keys maximum for 1 day
  }
}`, fixed: `
resource "kafka_topic" "topic_wrong_retention_comment" {
  name               = "topic_wrong_retention_comment"
  replication_factor = 3
  config = {
    "max.compaction.lag.ms" = "3600000" # allow not compacted keys maximum for 1 hour
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "max.compaction.lag.ms value doesn't correspond to the human readable value in the comment: fixing it ...",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 41},
					End:      hcl.Pos{Line: 7, Column: 1},
				},
			},
		},
	},
	{
		// the value is validated in the msk_topic_config rule
		name: "max compaction lag invalid",
		input: `
resource "kafka_topic" "topic_def" {
  name               = "topic_def"
  replication_factor = 3
  config = {
    "max.compaction.lag.ms" = "invalid-val"
  }
}`,
		expected: []*helper.Issue{
			{
				Message: "max.compaction.lag.ms must have a valid integer value expressed in milliseconds",
				Range: hcl.Range{
					Filename: fileName,
					Start:    hcl.Pos{Line: 6, Column: 31},
					End:      hcl.Pos{Line: 6, Column: 44},
				},
			},
		},
	},
}

func Test_MSKTopicConfigCommentsRule(t *testing.T) {
	rule := &MSKTopicConfigCommentsRule{}
	for _, tc := range configValueCommentsTests {
		t.Run(tc.name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{fileName: tc.input})
			require.NoError(t, rule.Check(runner))

			setExpectedRule(tc.expected, rule)
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
