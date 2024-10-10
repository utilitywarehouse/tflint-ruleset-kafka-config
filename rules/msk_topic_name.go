package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MskTopicNameRule checks whether a topic defined in MSK has the allowed team prefix.
type MskTopicNameRule struct {
	tflint.DefaultRule
}

func (r *MskTopicNameRule) Name() string {
	return "msk_topic_name"
}

func (r *MskTopicNameRule) Enabled() bool {
	return true
}

func (r *MskTopicNameRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MskTopicNameRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MskTopicNameRule) Check(runner tflint.Runner) error {
	path, err := runner.GetModulePath()
	if err != nil {
		return fmt.Errorf("getting module path: %w", err)
	}
	if !path.IsRoot() {
		logger.Debug("skipping child module")
		return nil
	}

	resourceContents, err := runner.GetResourceContent(
		"kafka_topic",
		&hclext.BodySchema{
			Attributes: []hclext.AttributeSchema{{Name: "name"}},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("getting kafka_topic contents: %w", err)
	}

	modulePath, err := runner.GetOriginalwd()
	if err != nil {
		return fmt.Errorf("failed getting module path: %w", err)
	}
	teamName := filepath.Base(modulePath)

	for _, topicResource := range resourceContents.Blocks {
		if err := r.validateTopic(runner, topicResource, teamName); err != nil {
			return err
		}
	}

	return nil
}

func (r *MskTopicNameRule) validateTopic(runner tflint.Runner, topic *hclext.Block, teamName string) error {
	resourceName := topic.Labels[1]
	nameAttr := topic.Body.Attributes["name"]

	var topicName string
	diags := gohcl.DecodeExpression(nameAttr.Expr, nil, &topicName)
	if diags.HasErrors() {
		return fmt.Errorf("decoding name for kafka_topic '%s': %w", resourceName, diags)
	}

	if !strings.HasPrefix(topicName, teamName+".") {
		err := runner.EmitIssue(
			r,
			fmt.Sprintf("topic name must have as a prefix the team name '%s'. Current value is '%s'", teamName, topicName),
			nameAttr.Range,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: topic name doesn't have the team prefix: %w", err)
		}
	}
	return nil
}
