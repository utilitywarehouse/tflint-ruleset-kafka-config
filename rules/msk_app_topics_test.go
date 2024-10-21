package rules

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/require"
	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func Test_MSKAppTopics(t *testing.T) {
	rule := &MSKAppTopics{}

	for _, tc := range []struct {
		name     string
		files    map[string]string
		expected helper.Issues
	}{
		{
			name: "consuming from topic not in module",
			files: map[string]string{
				"file.tf": `
module "consumer" {
	consume_topics = ["some_topic"]
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'consume_topics' may only contain topics defined in the current module but 'some_topic' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 33},
					},
				},
			},
		},
		{
			name: "producing from topic not in module",
			files: map[string]string{
				"file.tf": `
module "consumer" {
	produce_topics = ["some_topic"]
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'produce_topics' may only contain topics defined in the current module but 'some_topic' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 33},
					},
				},
			},
		},
		{
			name: "producing and consuming from topic not in module",
			files: map[string]string{
				"file.tf": `
module "consumer" {
	consume_topics = ["some_topic"]
	produce_topics = ["some_topic"]
}
`,
			},
			expected: []*helper.Issue{
				{
					Rule:    rule,
					Message: "'consume_topics' may only contain topics defined in the current module but 'some_topic' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 3, Column: 2},
						End:      hcl.Pos{Line: 3, Column: 33},
					},
				},
				{
					Rule:    rule,
					Message: "'produce_topics' may only contain topics defined in the current module but 'some_topic' is not",
					Range: hcl.Range{
						Filename: "file.tf",
						Start:    hcl.Pos{Line: 4, Column: 2},
						End:      hcl.Pos{Line: 4, Column: 33},
					},
				},
			},
		},
		{
			name: "external topic defined outside of consumer/producer",
			files: map[string]string{
				"file.tf": `
module "some_other_module" {
	kafka_stuff = ["external_topic"]
}
`,
			},
			expected: []*helper.Issue{},
		},
		{
			name: "all topics defined in file",
			files: map[string]string{
				"file.tf": `
resource "kafka_topic" "first_topic" {
	name = "first_topic"
}

resource "kafka_topic" "second_topic" {
	name = "second_topic"
}

module "consumer" {
	consume_topics = [kafka_topic.first_topic.name, kafka_topic.second_topic.name]
	produce_topics = [kafka_topic.first_topic.name]
}
`,
			},
			expected: []*helper.Issue{},
		},
		{
			name: "topics defined in separate file",
			files: map[string]string{
				"topics.tf": `
resource "kafka_topic" "first_topic" {
	name = "first_topic"
}

resource "kafka_topic" "second_topic" {
	name = "second_topic"
}
`,
				"file.tf": `
module "consumer" {
	consume_topics = [kafka_topic.first_topic.name, kafka_topic.second_topic.name]
	produce_topics = [kafka_topic.first_topic.name]
}
`,
			},
			expected: []*helper.Issue{},
		},
		{
			name: "topic name as string",
			files: map[string]string{
				"file.tf": `
resource "kafka_topic" "first_topic" {
	name = "my_topic"
}

module "consumer" {
	consume_topics = ["my_topic"]
}

# other resources in the module should be ignored
resource "some_resource" "some_other_resource" {
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
