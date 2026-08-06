package rules

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MSKTopicMaxMessageBytesRule checks that a topic's max.message.bytes, when specified, doesn't exceed the allowed limit.
type MSKTopicMaxMessageBytesRule struct {
	tflint.DefaultRule
}

func (r *MSKTopicMaxMessageBytesRule) Name() string {
	return "msk_topic_max_message_bytes"
}

func (r *MSKTopicMaxMessageBytesRule) Enabled() bool {
	return true
}

func (r *MSKTopicMaxMessageBytesRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKTopicMaxMessageBytesRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKTopicMaxMessageBytesRule) Check(runner tflint.Runner) error {
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
		if err := r.validateMaxMessageBytesForTopic(runner, topicResource); err != nil {
			return err
		}
	}

	return nil
}

const (
	maxMessageBytesAttr  = "max.message.bytes"
	maxMessageBytesLimit = 3 * 1024 * 1024 // 3MB
)

func (r *MSKTopicMaxMessageBytesRule) validateMaxMessageBytesForTopic(
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

	mmbPair, hasMmb := configKeyToPairMap[maxMessageBytesAttr]
	if !hasMmb {
		return nil
	}

	var mmbVal string
	diags := gohcl.DecodeExpression(mmbPair.Value, nil, &mmbVal)
	if diags.HasErrors() {
		return diags
	}

	mmbIntVal, err := strconv.Atoi(mmbVal)
	if err != nil {
		msg := fmt.Sprintf("%s must have a valid integer value expressed in bytes", maxMessageBytesAttr)
		err := runner.EmitIssue(r, msg, mmbPair.Value.Range())
		if err != nil {
			return fmt.Errorf("emitting issue: invalid max message bytes: %w", err)
		}
		return nil
	}

	if mmbIntVal > maxMessageBytesLimit {
		msg := fmt.Sprintf("%s must be less than or equal to %d bytes (3MB)", maxMessageBytesAttr, maxMessageBytesLimit)
		err := runner.EmitIssue(r, msg, mmbPair.Value.Range())
		if err != nil {
			return fmt.Errorf("emitting issue: max message bytes too large: %w", err)
		}
	}

	return nil
}
