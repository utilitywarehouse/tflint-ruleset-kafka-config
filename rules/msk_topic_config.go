package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MSKTopicConfigRule checks the configuration for an MSK topic.
type MSKTopicConfigRule struct {
	tflint.DefaultRule
}

func (r *MSKTopicConfigRule) Name() string {
	return "msk_topic_config"
}

func (r *MSKTopicConfigRule) Enabled() bool {
	return true
}

func (r *MSKTopicConfigRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKTopicConfigRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKTopicConfigRule) Check(runner tflint.Runner) error {
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
				{Name: replFactorAttrName},
				{Name: "config"},
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

func (r *MSKTopicConfigRule) validateTopicConfig(runner tflint.Runner, topic *hclext.Block) error {
	if err := r.validateReplicationFactor(runner, topic); err != nil {
		return err
	}

	configAttr, hasConfig := topic.Body.Attributes["config"]
	if !hasConfig {
		err := runner.EmitIssue(
			r,
			"missing config attribute: the topic configuration must be specified in a config attribute",
			topic.DefRange,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: missing config block: %w", err)
		}
		return nil
	}

	/* construct a mapping between the config key and the config KeyPair. This helps in both checking if a key is defined and to propose fixes to the values*/
	configKeyToPairMap, err := constructConfigKeyToPairMap(configAttr)
	if err != nil {
		return err
	}

	if err := r.validateCompressionType(runner, configAttr, configKeyToPairMap); err != nil {
		return err
	}

	if err := r.validateCleanupPolicy(runner, configAttr, configKeyToPairMap); err != nil {
		return err
	}
	return nil
}

func constructConfigKeyToPairMap(configAttr *hclext.Attribute) (map[string]hcl.KeyValuePair, error) {
	configExpr, ok := configAttr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("could not convert 'config' of type %T to hclsyntax.ObjectConsExpr", configExpr)
	}

	res := make(map[string]hcl.KeyValuePair, len(configExpr.ExprMap()))
	for _, pair := range configExpr.ExprMap() {
		var pk string
		diags := gohcl.DecodeExpression(pair.Key, nil, &pk)
		if diags.HasErrors() {
			return nil, diags
		}
		res[pk] = pair
	}
	return res, nil
}

const (
	replFactorAttrName = "replication_factor"
	// See [https://github.com/utilitywarehouse/tflint-ruleset-kafka-config/blob/main/rules/msk_topic_config.md#requirements] for explanation.
	replicationFactorVal = 3
)

var replFactorFix = fmt.Sprintf("%s = %d", replFactorAttrName, replicationFactorVal)

func (r *MSKTopicConfigRule) validateReplicationFactor(runner tflint.Runner, topic *hclext.Block) error {
	replFactorAttr, hasReplFactor := topic.Body.Attributes[replFactorAttrName]
	if !hasReplFactor {
		return r.reportMissingReplicationFactor(runner, topic)
	}

	var replFactor int
	diags := gohcl.DecodeExpression(replFactorAttr.Expr, nil, &replFactor)
	if diags.HasErrors() {
		return diags
	}

	if replFactor != replicationFactorVal {
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

func (r *MSKTopicConfigRule) reportMissingReplicationFactor(runner tflint.Runner, topic *hclext.Block) error {
	nameAttr, hasName := topic.Body.Attributes["name"]
	if !hasName {
		/*	when no name attribute, we can not issue a fix, as we insert the replication factor after the name */
		err := runner.EmitIssue(
			r,
			fmt.Sprintf("missing replication_factor: it must be equal to '%d'", replicationFactorVal),
			topic.DefRange,
		)
		if err != nil {
			return fmt.Errorf("emitting issue without fix: no replication factor: %w", err)
		}
		return nil
	}

	err := runner.EmitIssueWithFix(
		r,
		fmt.Sprintf("missing replication_factor: it must be equal to '%d'", replicationFactorVal),
		topic.DefRange,
		func(f tflint.Fixer) error {
			return f.InsertTextAfter(nameAttr.Range, "\n"+replFactorFix)
		},
	)
	if err != nil {
		return fmt.Errorf("emitting issue with fix: no replication factor: %w", err)
	}
	return nil
}

const (
	compressionTypeKey = "compression.type"
	compressionTypeVal = "zstd"
)

var compressionTypeFix = fmt.Sprintf(`"%s" = "%s"`, compressionTypeKey, compressionTypeVal)

func (r *MSKTopicConfigRule) validateCompressionType(
	runner tflint.Runner,
	config *hclext.Attribute,
	configPairMap map[string]hcl.KeyValuePair,
) error {
	ctPair, hasCt := configPairMap[compressionTypeKey]
	if !hasCt {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("missing %s: it must be equal to '%s'", compressionTypeKey, compressionTypeVal),
			config.Range,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(config.Expr.StartRange(), "\n"+compressionTypeFix)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue with fix: no compression type: %w", err)
		}
		return nil
	}

	var ctVal string
	diags := gohcl.DecodeExpression(ctPair.Value, nil, &ctVal)
	if diags.HasErrors() {
		return diags
	}

	if ctVal != compressionTypeVal {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("the %s value must be equal to '%s'", compressionTypeKey, compressionTypeVal),
			ctPair.Key.Range(),
			func(f tflint.Fixer) error {
				return f.ReplaceText(ctPair.Value.Range(), `"`+compressionTypeVal+`"`)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue with fix: wrong compression type: %w", err)
		}
		return nil
	}
	return nil
}

const (
	cleanupPolicyKey     = "cleanup.policy"
	cleanupPolicyDefault = "delete"
)

var (
	cleanupPolicyDefaultFix  = fmt.Sprintf(`"%s" = "%s"`, cleanupPolicyKey, cleanupPolicyDefault)
	cleanupPolicyValidValues = []string{"delete", "compact"}
)

func (r *MSKTopicConfigRule) validateCleanupPolicy(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	cpPair, hasCp := configKeyToPairMap[cleanupPolicyKey]
	if !hasCp {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("missing %s: using default '%s'", cleanupPolicyKey, cleanupPolicyDefault),
			config.Range,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(config.Expr.StartRange(), "\n"+cleanupPolicyDefaultFix)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue with fix: no cleanup policy: %w", err)
		}
		return nil
	}

	var cpVal string
	diags := gohcl.DecodeExpression(cpPair.Value, nil, &cpVal)
	if diags.HasErrors() {
		return diags
	}
	if !slices.Contains(cleanupPolicyValidValues, cpVal) {
		err := runner.EmitIssue(
			r,
			fmt.Sprintf(
				"invalid %s: it must be one of [%s], but currently is '%s'",
				cleanupPolicyKey,
				strings.Join(cleanupPolicyValidValues, ", "),
				cpVal,
			),
			cpPair.Value.Range(),
		)
		if err != nil {
			return fmt.Errorf("emitting issue: invalid cleanup policy: %w", err)
		}
		return nil
	}
	return nil
}
