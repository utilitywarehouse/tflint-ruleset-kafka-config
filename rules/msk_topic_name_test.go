package rules

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MskTopics(t *testing.T) {
	rule := &MskTopicNameRule{}

	for _, tc := range []struct {
		name     string
		files    map[string]string
		workDir  string
		expected helper.Issues
	}{
		{
			name:    "topic doesn't contain the team prefix",
			workDir: filepath.Join("kafka-cluster-config", "dev-aws", "kafka-shared-msk", "pubsub"),
			files: map[string]string{
				"topics.tf": `
resource "kafka_topic" "wrong_topic" {
	name = "name-without-prefix"
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "topic name must be prefixed with the team name 'pubsub'. Current value is 'name-without-prefix'",
					Range: hcl.Range{
						Filename: "topics.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 30},
					},
				},
			},
		},
		{
			name:    "topic doesn't have alias as prefix",
			workDir: filepath.Join("kafka-cluster-config", "dev-aws", "kafka-shared-msk", "pubsub"),
			files: map[string]string{
				".tflint.hcl": `
rule "msk_topic_name" {
  enabled = true
  team_aliases = {
	pubsub = ["alias_pubsub1", "alias_pubsub2"]
	otel = ["alias_otel"]
  }
}`,
				"topics.tf": `
resource "kafka_topic" "wrong_topic" {
	name = "name-without-prefix"
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "topic name must be prefixed with the team name 'pubsub' or one of its aliases 'alias_pubsub1, alias_pubsub2'. Current value is 'name-without-prefix'",
					Range: hcl.Range{
						Filename: "topics.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 30},
					},
				},
			},
		},
		{
			name:    "good topic with prefix as alias from config",
			workDir: filepath.Join("kafka-cluster-config", "dev-aws", "kafka-shared-msk", "pubsub"),
			files: map[string]string{
				".tflint.hcl": `
rule "msk_topic_name" {
  enabled = true
  team_aliases = {
	pubsub = ["alias_pubsub1", "alias_pubsub2"]
  }
}`,
				"topics.tf": `
resource "kafka_topic" "good_topic_from_alias_1" {
	name = "alias_pubsub1.good-topic"
}
resource "kafka_topic" "good_topic_from_alias_2" {
	name = "alias_pubsub2.good-topic"
}
`,
			},
			expected: []*helper.Issue{},
		},
		{
			name:    "good topic definition with team name prefix",
			workDir: filepath.Join("kafka-cluster-config", "dev-aws", "kafka-shared-msk", "pubsub"),
			files: map[string]string{
				"topics.tf": `
resource "kafka_topic" "good_topic" {
	name = "pubsub.good-topic"
}
`,
			},
			expected: []*helper.Issue{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := WithWorkDir(helper.TestRunner(t, tc.files), tc.workDir)

			require.NoError(t, rule.Check(runner))

			helper.AssertIssues(t, tc.expected, runner.Issues)
		})
	}
}
