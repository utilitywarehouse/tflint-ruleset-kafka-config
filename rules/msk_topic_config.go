package rules

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MskTopicConfigRule checks the configuration for an MSK topic.
type MskTopicConfigRule struct {
	tflint.DefaultRule
}

func (r *MskTopicConfigRule) Name() string {
	return "msk_topic_config"
}

func (r *MskTopicConfigRule) Enabled() bool {
	return true
}

func (r *MskTopicConfigRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MskTopicConfigRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MskTopicConfigRule) Check(runner tflint.Runner) error {
	path, err := runner.GetModulePath()
	if err != nil {
		return fmt.Errorf("getting module path: %w", err)
	}
	if !path.IsRoot() {
		logger.Debug("skipping child module")
		return nil
	}

	/*
		resource "kafka_topic" "proximo_example" {
		  name               = "pubsub.proximo-example"
		  replication_factor = 3
		  partitions         = 3
		  config = {
		    # retain 100MB on each partition
		    "retention.bytes" = "104857600"
		    # keep data for 2 days
		    "retention.ms" = "172800000"
		    # allow max 1 MB for a message
		    "max.message.bytes" = "1048576"
		    "compression.type"  = "zstd"
		    "cleanup.policy"    = "delete"
		  }
		}
	*/
	resourceContents, err := runner.GetResourceContent(
		"kafka_topic",
		&hclext.BodySchema{
			Attributes: []hclext.AttributeSchema{
				{Name: "name"},
				{Name: "replication_factor"},
			},
			Blocks: []hclext.BlockSchema{
				{
					Type: "config",
					Body: &hclext.BodySchema{
						Attributes: []hclext.AttributeSchema{
							{Name: "retention.ms"},
							{Name: "compression.type"},
							{Name: "cleanup.policy"},
						},
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("getting kafka_topic contents: %w", err)
	}

	for _, topicResource := range resourceContents.Blocks {
		if err := r.validateTopicConfig(runner, topicResource); err != nil {
			return err
		}
	}

	return nil
}

func (r *MskTopicConfigRule) validateTopicConfig(runner tflint.Runner, topic *hclext.Block) error {
	// resourceName := topic.Labels[1]

	nameAttr := topic.Body.Attributes["name"]
	if err := r.validateReplicationFactor(runner, topic, nameAttr); err != nil {
		return err
	}

	return nil
}

const (
	replFactorAttrName   = "replication_factor"
	replicationFactorVal = 3
)

var replFactorFix = fmt.Sprintf("%s = %d", replFactorAttrName, replicationFactorVal)

func (r *MskTopicConfigRule) validateReplicationFactor(
	runner tflint.Runner,
	topic *hclext.Block,
	nameAttr *hclext.Attribute,
) error {
	replFactorAttr, hasReplFactor := topic.Body.Attributes[replFactorAttrName]
	if !hasReplFactor {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("missing replication_factor: it must be equal to '%d'", replicationFactorVal),
			topic.DefRange,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(nameAttr.Range, "\n"+replFactorFix)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: no replication factor: %w", err)
		}
		return nil
	}

	var replFactor int
	diags := gohcl.DecodeExpression(replFactorAttr.Expr, nil, &replFactor)
	if diags.HasErrors() {
		return diags
	}

	if replFactor != 3 {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("the replication_factor must be equal to '%d'", replicationFactorVal),
			replFactorAttr.Range,
			func(f tflint.Fixer) error {
				return f.ReplaceText(replFactorAttr.Range, replFactorFix)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: incorrect replication factor: %w", err)
		}
	}
	return nil
}
