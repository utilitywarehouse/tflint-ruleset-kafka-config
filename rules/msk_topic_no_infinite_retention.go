package rules

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MSKTopicNoInfiniteRetentionRule checks that a topic doesn't define infinite retention.
// Defined in a separate rule than the MSKTopicConfigRule, as we allow this one to be ignored.
type MSKTopicNoInfiniteRetentionRule struct {
	tflint.DefaultRule
}

const ruleName = "msk_topic_no_infinite_retention"

func (r *MSKTopicNoInfiniteRetentionRule) Name() string {
	return ruleName
}

func (r *MSKTopicNoInfiniteRetentionRule) Enabled() bool {
	return true
}

func (r *MSKTopicNoInfiniteRetentionRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKTopicNoInfiniteRetentionRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKTopicNoInfiniteRetentionRule) Check(runner tflint.Runner) error {
	isRoot, err := isRootModule(runner)
	if err != nil {
		return err
	}
	if !isRoot {
		logger.Debug("skipping child module")
		return nil
	}

	resourceContents, err := runner.GetResourceContent(
		"kafka_topic",
		&hclext.BodySchema{
			Attributes: []hclext.AttributeSchema{
				{Name: "name"},
				{Name: "config"},
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("getting kafka_topic contents: %w", err)
	}

	for _, topicResource := range resourceContents.Blocks {
		if err := r.validateNoInfiniteRetentionForTopic(runner, topicResource); err != nil {
			return err
		}
	}

	return nil
}

var infiniteRetentionMsg = fmt.Sprintf(
	`Warning: Infinite retention is NOT recommended. Please check if a compacted topic fits your use case. Otherwise consider using databases for long term storage.
Until you fix this issue, please suppress this rule, stating the reason why infinite retention is needed, by putting a comment like this on the 'retention.ms' property: 
# tflint-ignore: %s, # infinite retention because ...
"retention.ms"    = "-1"`,
	ruleName,
)

func (r *MSKTopicNoInfiniteRetentionRule) validateNoInfiniteRetentionForTopic(
	runner tflint.Runner,
	topic *hclext.Block,
) error {
	configAttr, hasConfig := topic.Body.Attributes["config"]
	if !hasConfig {
		return nil
	}

	/* construct a mapping between the config key and the config KeyPair. This helps in both checking if a key is defined and to propose fixes to the values*/
	configKeyToPairMap, err := constructConfigKeyToPairMap(configAttr)
	if err != nil {
		return err
	}

	retTimePair, hasRetTime := configKeyToPairMap[retentionTimeAttr]
	if !hasRetTime {
		return nil
	}

	var retTimeVal string
	diags := gohcl.DecodeExpression(retTimePair.Value, nil, &retTimeVal)
	if diags.HasErrors() {
		return fmt.Errorf("evaluating retention time: %w", diags)
	}

	retTimeIntVal, err := strconv.Atoi(retTimeVal)
	if err != nil {
		//nolint:nilerr // just exit, this is handled in the topic config rule
		return nil
	}

	if isInfiniteRetention(retTimeIntVal) {
		err := runner.EmitIssue(r, infiniteRetentionMsg, retTimePair.Value.Range())
		if err != nil {
			return fmt.Errorf("emitting issue: infinite retention: %w", err)
		}
		return nil
	}

	return nil
}
