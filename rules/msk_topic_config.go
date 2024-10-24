package rules

import (
	"fmt"
	"slices"
	"strconv"
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

	cleanupPolicy, err := r.getAndValidateCleanupPolicy(runner, configAttr, configKeyToPairMap)
	if err != nil {
		return err
	}
	switch cleanupPolicy {
	case cleanupPolicyDelete:
		if err := r.validateRetentionForDeletePolicy(runner, configAttr, configKeyToPairMap); err != nil {
			return err
		}
	case cleanupPolicyCompact:
		// todo: validate no retention & remote storage for compact
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
			ctPair.Value.Range(),
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
	cleanupPolicyDelete  = "delete"
	cleanupPolicyCompact = "compact"
	cleanupPolicyDefault = cleanupPolicyDelete
)

var (
	cleanupPolicyDefaultFix  = fmt.Sprintf(`"%s" = "%s"`, cleanupPolicyKey, cleanupPolicyDefault)
	cleanupPolicyValidValues = []string{cleanupPolicyDelete, cleanupPolicyCompact}
)

func (r *MSKTopicConfigRule) getAndValidateCleanupPolicy(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) (string, error) {
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
			return "", fmt.Errorf("emitting issue with fix: no cleanup policy: %w", err)
		}
		return cleanupPolicyDefault, nil
	}

	var cpVal string
	diags := gohcl.DecodeExpression(cpPair.Value, nil, &cpVal)
	if diags.HasErrors() {
		return "", diags
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
			return "", fmt.Errorf("emitting issue: invalid cleanup policy: %w", err)
		}
		return "", nil
	}
	return cpVal, nil
}

const (
	retentionTimeAttr = "retention.ms"
	millisInOneDay    = 1 * 24 * 60 * 60 * 1000
	// The threshold on retention time when remote storage is supported.
	tieredStorageThresholdInDays    = 3
	tieredStorageEnableAttr         = "remote.storage.enable"
	tieredStorageEnabledValue       = "true"
	localRetentionTimeAttr          = "local.retention.ms"
	localRetentionTimeInDaysDefault = 1
)

/*	Putting an invalid value by default to force users to put a valid value */
var (
	retentionTimeDefTemplate = fmt.Sprintf(`"%s" = "???"`, retentionTimeAttr)
	enableTieredStorage      = fmt.Sprintf(`"%s" = "%s"`, tieredStorageEnableAttr, tieredStorageEnabledValue)
	localRetentionTimeFix    = fmt.Sprintf(
		`# keep data in hot storage for %d day
     "%s" = "%d"`,
		localRetentionTimeInDaysDefault,
		localRetentionTimeAttr,
		localRetentionTimeInDaysDefault*millisInOneDay)
)

func (r *MSKTopicConfigRule) validateRetentionForDeletePolicy(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	rtIntVal, err := r.getAndValidateRetentionTime(runner, config, configKeyToPairMap)
	if err != nil || rtIntVal == nil {
		return err
	}

	if *rtIntVal <= tieredStorageThresholdInDays*millisInOneDay && !isInfiniteRetention(*rtIntVal) {
		return nil
	}

	err = r.validateTieredStorageEnabled(runner, config, configKeyToPairMap)
	if err != nil {
		return err
	}

	_, hasLocalRetTimeAttr := configKeyToPairMap[localRetentionTimeAttr]
	if !hasLocalRetTimeAttr {
		msg := fmt.Sprintf(
			"missing %s when tiered storage is enabled: using default '%d'",
			localRetentionTimeAttr,
			localRetentionTimeInDaysDefault*millisInOneDay,
		)
		err := runner.EmitIssueWithFix(r, msg, config.Range,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(config.Expr.StartRange(), "\n"+localRetentionTimeFix)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: remote storage enable: %w", err)
		}
	}
	return nil
}

func isInfiniteRetention(rtIntVal int) bool {
	return rtIntVal < 0
}

func (r *MSKTopicConfigRule) validateTieredStorageEnabled(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	tieredStoragePair, hasTieredStorageAttr := configKeyToPairMap[tieredStorageEnableAttr]
	if !hasTieredStorageAttr {
		msg := fmt.Sprintf(
			"tiered storage should be enabled when retention time is longer than %d days",
			tieredStorageThresholdInDays,
		)
		err := runner.EmitIssueWithFix(r, msg, config.Range,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(config.Expr.StartRange(), "\n"+enableTieredStorage)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: remote storage enable: %w", err)
		}
		return nil
	}

	var tieredStorageVal string
	diags := gohcl.DecodeExpression(tieredStoragePair.Value, nil, &tieredStorageVal)
	if diags.HasErrors() {
		return diags
	}

	if tieredStorageVal != tieredStorageEnabledValue {
		msg := fmt.Sprintf(
			"tiered storage should be enabled when retention time is longer than %d days",
			tieredStorageThresholdInDays,
		)
		err := runner.EmitIssueWithFix(r, msg, tieredStoragePair.Value.Range(),
			func(f tflint.Fixer) error {
				return f.ReplaceText(tieredStoragePair.Value.Range(), fmt.Sprintf(`"%s"`, tieredStorageEnabledValue))
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: set remote storage on enable: %w", err)
		}
	}

	return nil
}

func (r *MSKTopicConfigRule) getAndValidateRetentionTime(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) (*int, error) {
	rtPair, hasRt := configKeyToPairMap[retentionTimeAttr]
	if !hasRt {
		msg := fmt.Sprintf("%s must be defined on a topic with cleanup policy delete", retentionTimeAttr)
		err := runner.EmitIssueWithFix(r, msg, config.Range,
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(config.Expr.StartRange(), "\n"+retentionTimeDefTemplate)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("emitting issue: no retention time: %w", err)
		}
		return nil, nil
	}

	var rtVal string
	diags := gohcl.DecodeExpression(rtPair.Value, nil, &rtVal)
	if diags.HasErrors() {
		return nil, diags
	}

	rtIntVal, err := strconv.Atoi(rtVal)
	if err != nil {
		msg := fmt.Sprintf(
			"%s must have a valid integer value expressed in milliseconds. Use -1 for infinite retention",
			retentionTimeAttr,
		)
		err := runner.EmitIssue(r, msg, rtPair.Value.Range())
		if err != nil {
			return nil, fmt.Errorf("emitting issue: invalid retention time: %w", err)
		}
		return nil, nil
	}
	return &rtIntVal, nil
}
