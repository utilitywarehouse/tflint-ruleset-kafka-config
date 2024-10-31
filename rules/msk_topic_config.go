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

	configAttr, err := r.validateAndGetConfigAttr(runner, topic)
	if err != nil {
		return err
	}

	if configAttr == nil {
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

	if err = r.validateCleanupPolicyConfig(runner, configAttr, configKeyToPairMap); err != nil {
		return err
	}

	if err = r.validateConfigValuesInComments(runner, configKeyToPairMap); err != nil {
		return err
	}
	return nil
}

func (r *MSKTopicConfigRule) validateCleanupPolicyConfig(
	runner tflint.Runner,
	configAttr *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	cleanupPolicy, err := r.getAndValidateCleanupPolicyValue(runner, configAttr, configKeyToPairMap)
	if err != nil {
		return err
	}

	switch cleanupPolicy {
	case cleanupPolicyDelete:
		if err := r.validateRetentionForDeletePolicy(runner, configAttr, configKeyToPairMap); err != nil {
			return err
		}
	case cleanupPolicyCompact:
		reason := "compacted topic"
		if err := r.validateTieredStorageDisabled(runner, configKeyToPairMap, reason); err != nil {
			return err
		}
		if err := r.validateLocalRetentionNotDefined(runner, configKeyToPairMap, reason); err != nil {
			return err
		}
		if err := r.validateRetentionTimeNotDefined(runner, configKeyToPairMap, reason); err != nil {
			return err
		}
	}
	return nil
}

func (r *MSKTopicConfigRule) validateAndGetConfigAttr(
	runner tflint.Runner,
	topic *hclext.Block,
) (*hclext.Attribute, error) {
	configAttr, hasConfig := topic.Body.Attributes["config"]
	if !hasConfig {
		err := runner.EmitIssue(
			r,
			"missing config attribute: the topic configuration must be specified in a config attribute",
			topic.DefRange,
		)
		if err != nil {
			return nil, fmt.Errorf("emitting issue: missing config block: %w", err)
		}
		return nil, nil
	}
	return configAttr, nil
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

func (r *MSKTopicConfigRule) getAndValidateCleanupPolicyValue(
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
	millisInOneHour   = 60 * 60 * 1000
	millisInOneDay    = 24 * millisInOneHour
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
	retentionTime, err := r.getAndValidateRetentionTime(runner, config, configKeyToPairMap)
	if err != nil {
		return err
	}

	if retentionTime == nil {
		return nil
	}

	if mustEnableTieredStorage(*retentionTime) {
		if err := r.validateTieredStorageEnabled(runner, config, configKeyToPairMap); err != nil {
			return err
		}

		if err := r.validateLocalRetentionDefined(runner, config, configKeyToPairMap); err != nil {
			return err
		}
	} else {
		reason := fmt.Sprintf("less than %d days retention", tieredStorageThresholdInDays)
		if err := r.validateTieredStorageDisabled(runner, configKeyToPairMap, reason); err != nil {
			return err
		}

		if err := r.validateLocalRetentionNotDefined(runner, configKeyToPairMap, reason); err != nil {
			return err
		}
	}

	return nil
}

func mustEnableTieredStorage(retentionTime int) bool {
	return retentionTime >= tieredStorageThresholdInDays*millisInOneDay || isInfiniteRetention(retentionTime)
}

func (r *MSKTopicConfigRule) validateLocalRetentionDefined(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	localRetTimePair, hasLocalRetTimeAttr := configKeyToPairMap[localRetentionTimeAttr]
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
		return nil
	}

	var localRetTimeVal string
	diags := gohcl.DecodeExpression(localRetTimePair.Value, nil, &localRetTimeVal)
	if diags.HasErrors() {
		return diags
	}

	_, err := strconv.Atoi(localRetTimeVal)
	if err != nil {
		msg := fmt.Sprintf(
			"%s must have a valid integer value expressed in milliseconds",
			localRetentionTimeAttr,
		)
		err := runner.EmitIssue(r, msg, localRetTimePair.Value.Range())
		if err != nil {
			return fmt.Errorf("emitting issue: invalid local retention time: %w", err)
		}
		return nil
	}

	return nil
}

func (r *MSKTopicConfigRule) validateLocalRetentionNotDefined(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
	reason string,
) error {
	localRetTimePair, hasLocalRetTimeAttr := configKeyToPairMap[localRetentionTimeAttr]
	if !hasLocalRetTimeAttr {
		return nil
	}

	msg := fmt.Sprintf(
		"defining %s is misleading when tiered storage is disabled due to %s: removing it...",
		localRetentionTimeAttr,
		reason,
	)
	err := runner.EmitIssueWithFix(r, msg, localRetTimePair.Value.Range(),
		func(f tflint.Fixer) error {
			/* remove the whole key + value */
			keyRange := localRetTimePair.Key.Range()
			return f.Remove(
				hcl.Range{
					Filename: keyRange.Filename,
					Start:    keyRange.Start,
					End:      localRetTimePair.Value.Range().End,
				},
			)
		},
	)
	if err != nil {
		return fmt.Errorf("emitting issue: local storage specified for disabled tiered storage : %w", err)
	}
	return nil
}

func isInfiniteRetention(val int) bool {
	return val < 0
}

func (r *MSKTopicConfigRule) validateTieredStorageEnabled(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	tieredStoragePair, hasTieredStorageAttr := configKeyToPairMap[tieredStorageEnableAttr]
	tieredStorageEnableMsg := fmt.Sprintf(
		"tiered storage must be enabled when retention time is longer than %d days",
		tieredStorageThresholdInDays,
	)

	if !hasTieredStorageAttr {
		err := runner.EmitIssueWithFix(r, tieredStorageEnableMsg, config.Range,
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
		err := runner.EmitIssueWithFix(r, tieredStorageEnableMsg, tieredStoragePair.Value.Range(),
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

func (r *MSKTopicConfigRule) validateTieredStorageDisabled(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
	reason string,
) error {
	tieredStoragePair, hasTieredStorageAttr := configKeyToPairMap[tieredStorageEnableAttr]

	if !hasTieredStorageAttr {
		return nil
	}

	var tieredStorageVal string
	diags := gohcl.DecodeExpression(tieredStoragePair.Value, nil, &tieredStorageVal)
	if diags.HasErrors() {
		return diags
	}

	if tieredStorageVal != tieredStorageEnabledValue {
		return nil
	}

	msg := fmt.Sprintf(
		"tiered storage is not supported for %s: disabling it...",
		reason,
	)
	err := runner.EmitIssueWithFix(r, msg, tieredStoragePair.Value.Range(),
		func(f tflint.Fixer) error {
			/* remove the whole key + value */
			keyRange := tieredStoragePair.Key.Range()
			return f.Remove(
				hcl.Range{
					Filename: keyRange.Filename,
					Start:    keyRange.Start,
					End:      tieredStoragePair.Value.Range().End,
				},
			)
		},
	)
	if err != nil {
		return fmt.Errorf("emitting issue: remote storage enable: %w", err)
	}

	return nil
}

func (r *MSKTopicConfigRule) getAndValidateRetentionTime(
	runner tflint.Runner,
	config *hclext.Attribute,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) (*int, error) {
	retTimePair, hasRetTime := configKeyToPairMap[retentionTimeAttr]
	if !hasRetTime {
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

	var retTimeVal string
	diags := gohcl.DecodeExpression(retTimePair.Value, nil, &retTimeVal)
	if diags.HasErrors() {
		return nil, diags
	}

	retTimeIntVal, err := strconv.Atoi(retTimeVal)
	if err != nil {
		msg := fmt.Sprintf(
			"%s must have a valid integer value expressed in milliseconds. Use -1 for infinite retention",
			retentionTimeAttr,
		)
		err := runner.EmitIssue(r, msg, retTimePair.Value.Range())
		if err != nil {
			return nil, fmt.Errorf("emitting issue: invalid retention time: %w", err)
		}
		return nil, nil
	}
	return &retTimeIntVal, nil
}

func (r *MSKTopicConfigRule) validateRetentionTimeNotDefined(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
	reason string,
) error {
	retTimePair, hasRetTime := configKeyToPairMap[retentionTimeAttr]
	if !hasRetTime {
		return nil
	}
	msg := fmt.Sprintf("defining %s is misleading for %s: removing it...", retentionTimeAttr, reason)
	keyRange := retTimePair.Key.Range()

	err := runner.EmitIssueWithFix(r, msg, keyRange,
		func(f tflint.Fixer) error {
			return f.Remove(
				hcl.Range{
					Filename: keyRange.Filename,
					Start:    keyRange.Start,
					End:      retTimePair.Value.Range().End,
				},
			)
		},
	)
	if err != nil {
		return fmt.Errorf("emitting issue: retention time defined for compacted topic: %w", err)
	}
	return nil
}

func (r *MSKTopicConfigRule) validateConfigValuesInComments(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	retTimePair, hasRetTime := configKeyToPairMap[retentionTimeAttr]
	if !hasRetTime {
		return nil
	}

	msg, err := buildDurationComment(retTimePair, "-1")
	if err != nil {
		return err
	}
	if msg == "" {
		return nil
	}

	comment, err := r.getExistingComment(runner, retTimePair)
	if err != nil {
		return err
	}

	if comment == nil {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("%s must have a comment with the human readable value: adding it ...", retentionTimeAttr),
			retTimePair.Key.Range(),
			func(f tflint.Fixer) error {
				return f.InsertTextBefore(retTimePair.Key.Range(), msg+"\n")
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: incorrect replication factor: %w", err)
		}
		return nil
	}

	commentTxt := strings.TrimSpace(string(comment.Bytes))
	if commentTxt != msg {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf(
				"%s value doesn't correspond to the human readable value in the comment: fixing it ...",
				retentionTimeAttr,
			),
			comment.Range,
			func(f tflint.Fixer) error {
				return f.ReplaceText(comment.Range, msg+"\n")
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: incorrect replication factor: %w", err)
		}
	}

	return nil
}

func (r *MSKTopicConfigRule) getExistingComment(runner tflint.Runner, pair hcl.KeyValuePair) (*hclsyntax.Token, error) {
	comments, err := getCommentsForFile(runner, pair.Key.Range().Filename)
	if err != nil {
		return nil, err
	}

	// todo: check to use binary search
	idx := slices.IndexFunc(comments, func(comment hclsyntax.Token) bool {
		return comment.Range.End.Line == pair.Key.Range().Start.Line
	})

	if idx >= 0 {
		return &comments[idx], nil
	}
	return nil, nil
}

func getCommentsForFile(runner tflint.Runner, filename string) (hclsyntax.Tokens, error) {
	// todo: optimise this, as we're reading the file for each topic
	file, err := runner.GetFile(filename)
	if err != nil {
		return nil, fmt.Errorf("getting hcl file %s for reading comments: %w", filename, err)
	}

	tokens, diags := hclsyntax.LexConfig(file.Bytes, filename, hcl.InitialPos)
	if diags != nil {
		return nil, diags
	}

	isNotCommentFunc := func(token hclsyntax.Token) bool {
		return token.Type != hclsyntax.TokenComment
	}

	return slices.DeleteFunc(tokens, isNotCommentFunc), nil
}

func buildDurationComment(timePair hcl.KeyValuePair, infiniteVal string) (string, error) {
	var timeVal string
	diags := gohcl.DecodeExpression(timePair.Value, nil, &timeVal)
	if diags.HasErrors() {
		return "", diags
	}
	baseMsg := "keep data"

	if timeVal == infiniteVal {
		return fmt.Sprintf("# %s forever", baseMsg), nil
	}

	timeMillis, err := strconv.Atoi(timeVal)
	// todo: check what we should do here
	if err != nil {
		//nolint:nilerr
		return "", nil
	}

	timeUnits, unit := determineTimeUnits(timeMillis)

	msg := fmt.Sprintf("# %s for %d %s", baseMsg, timeUnits, unit)
	return msg, nil
}

func determineTimeUnits(millis int) (int, string) {
	timeInDays := millis / millisInOneDay

	if timeInDays > 0 {
		if timeInDays == 1 {
			return 1, "day"
		}
		return timeInDays, "days"
	}

	timeInHours := millis / millisInOneHour
	if timeInHours == 1 {
		return 1, "hour"
	}
	return timeInHours, "hours"
}
