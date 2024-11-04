package rules

import (
	"fmt"
	"math"
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

// MSKTopicConfigCommentsRule checks comments on time and bytes values.
type MSKTopicConfigCommentsRule struct {
	tflint.DefaultRule
}

func (r *MSKTopicConfigCommentsRule) Name() string {
	return "msk_topic_config_comments"
}

func (r *MSKTopicConfigCommentsRule) Enabled() bool {
	return true
}

func (r *MSKTopicConfigCommentsRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKTopicConfigCommentsRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKTopicConfigCommentsRule) Check(runner tflint.Runner) error {
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
				{Name: "config"},
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("getting kafka_topic contents: %w", err)
	}

	for _, topicResource := range resourceContents.Blocks {
		if err := r.validateTopicConfigComments(runner, topicResource); err != nil {
			return err
		}
	}

	return nil
}

func (r *MSKTopicConfigCommentsRule) validateTopicConfigComments(runner tflint.Runner, topic *hclext.Block) error {
	configAttr, hasConfig := topic.Body.Attributes["config"]
	if !hasConfig {
		return nil
	}

	/* construct a mapping between the config key and the config KeyPair. This helps in both checking if a key is defined and to propose fixes to the values*/
	configKeyToPairMap, err := constructConfigKeyToPairMap(configAttr)
	if err != nil {
		return err
	}

	if err = r.validateConfigValuesInComments(runner, configKeyToPairMap); err != nil {
		return err
	}
	return nil
}

type configTimeValueCommentInfo struct {
	key              string
	infiniteValue    string
	baseComment      string
	issueWhenInvalid bool
}

var configTimeValueCommentInfos = []configTimeValueCommentInfo{
	{
		key:              retentionTimeAttr,
		infiniteValue:    "-1",
		baseComment:      "keep data",
		issueWhenInvalid: false,
	},
	{
		key:              localRetentionTimeAttr,
		infiniteValue:    "-2",
		baseComment:      localRetentionTimeCommentBase,
		issueWhenInvalid: false,
	},
	{
		key:              "max.compaction.lag.ms",
		infiniteValue:    "",
		baseComment:      "allow not compacted keys maximum",
		issueWhenInvalid: true,
	},
}

func (r *MSKTopicConfigCommentsRule) validateConfigValuesInComments(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
) error {
	for _, configValueInfo := range configTimeValueCommentInfos {
		if err := r.validateTimeConfigValue(runner, configKeyToPairMap, configValueInfo); err != nil {
			return err
		}
	}

	return nil
}

func (r *MSKTopicConfigCommentsRule) validateTimeConfigValue(
	runner tflint.Runner,
	configKeyToPairMap map[string]hcl.KeyValuePair,
	configValueInfo configTimeValueCommentInfo,
) error {
	timePair, hasConfig := configKeyToPairMap[configValueInfo.key]
	if !hasConfig {
		return nil
	}

	msg, err := r.buildDurationComment(runner, timePair, configValueInfo)
	if err != nil {
		return err
	}
	if msg == "" {
		return nil
	}

	comment, err := r.getExistingComment(runner, timePair)
	if err != nil {
		return err
	}

	if comment == nil {
		err := runner.EmitIssueWithFix(
			r,
			fmt.Sprintf("%s must have a comment with the human readable value: adding it ...", configValueInfo.key),
			timePair.Key.Range(),
			func(f tflint.Fixer) error {
				return f.InsertTextAfter(timePair.Value.Range(), msg)
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: no comment for time value: %w", err)
		}
		return nil
	}

	commentTxt := strings.TrimSpace(string(comment.Bytes))
	if commentTxt != msg {
		issueMsg := fmt.Sprintf(
			"%s value doesn't correspond to the human readable value in the comment: fixing it ...",
			configValueInfo.key,
		)
		err := runner.EmitIssueWithFix(r, issueMsg, comment.Range,
			func(f tflint.Fixer) error {
				return f.ReplaceText(comment.Range, msg+"\n")
			},
		)
		if err != nil {
			return fmt.Errorf("emitting issue: wrong comment for time value: %w", err)
		}
	}
	return nil
}

func (r *MSKTopicConfigCommentsRule) getExistingComment(
	runner tflint.Runner,
	pair hcl.KeyValuePair,
) (*hclsyntax.Token, error) {
	comments, err := r.getCommentsForFile(runner, pair.Key.Range().Filename)
	if err != nil {
		return nil, err
	}

	// first look for the comment on the same line, after the property definition.
	// Example: "retention.ms" = "2629800000" # keep data for 30 days
	afterPropertyIdx := slices.IndexFunc(comments, func(comment hclsyntax.Token) bool {
		return comment.Range.Start.Line == pair.Key.Range().Start.Line &&
			comment.Range.Start.Column > pair.Value.Range().End.Column
	})

	if afterPropertyIdx >= 0 {
		return &comments[afterPropertyIdx], nil
	}

	/* second, look for the comment on the previous line, before the property definition. Example:
	# keep data for 30 days
	"retention.ms" = "2629800000"
	*/
	beforePropertyIdx := slices.IndexFunc(comments, func(comment hclsyntax.Token) bool {
		return comment.Range.Start.Line == pair.Key.Range().Start.Line-1 &&
			comment.Range.End.Line == pair.Key.Range().Start.Line
	})
	if beforePropertyIdx >= 0 {
		return &comments[beforePropertyIdx], nil
	}

	return nil, nil
}

func (r *MSKTopicConfigCommentsRule) getCommentsForFile(
	runner tflint.Runner,
	filename string,
) (hclsyntax.Tokens, error) {
	// we need to parse the file every time, otherwise keeping a cache per file doesn't work
	file, err := runner.GetFile(filename)
	if err != nil {
		return nil, fmt.Errorf("getting hcl file %s for reading comments: %w", filename, err)
	}

	tokens, diags := hclsyntax.LexConfig(file.Bytes, filename, hcl.InitialPos)
	if diags != nil {
		return nil, diags
	}

	return slices.DeleteFunc(tokens, isNotComment), nil
}

func isNotComment(token hclsyntax.Token) bool {
	return token.Type != hclsyntax.TokenComment
}

func (r *MSKTopicConfigCommentsRule) buildDurationComment(
	runner tflint.Runner,
	timePair hcl.KeyValuePair,
	configValueInfo configTimeValueCommentInfo,
) (string, error) {
	var timeVal string
	diags := gohcl.DecodeExpression(timePair.Value, nil, &timeVal)
	if diags.HasErrors() {
		return "", diags
	}

	if timeVal == configValueInfo.infiniteValue {
		return fmt.Sprintf("# %s forever", configValueInfo.baseComment), nil
	}

	timeMillis, err := strconv.Atoi(timeVal)
	if err != nil {
		if configValueInfo.issueWhenInvalid {
			issueMsg := fmt.Sprintf(
				"%s must have a valid integer value expressed in milliseconds",
				configValueInfo.key,
			)
			err := runner.EmitIssue(r, issueMsg, timePair.Value.Range())
			if err != nil {
				return "", fmt.Errorf("emitting issue: invalid time value: %w", err)
			}
		}

		return "", nil
	}

	baseComment := configValueInfo.baseComment

	msg := buildCommentForMillis(timeMillis, baseComment)
	return msg, nil
}

func buildCommentForMillis(timeMillis int, baseComment string) string {
	timeUnits, unit := determineTimeUnits(timeMillis)

	timeUnitsStr := strconv.FormatFloat(timeUnits, 'f', -1, 64)
	msg := fmt.Sprintf("# %s for %s %s", baseComment, timeUnitsStr, unit)
	return msg
}

/*	round to 1 digit precision  */
func round(val float64) float64 {
	return math.Round(val*10) / 10
}

func determineTimeUnits(millis int) (float64, string) {
	floatMillis := float64(millis)
	timeInYears := round(floatMillis / millisInOneYear)
	if timeInYears >= 1 {
		if timeInYears == 1 {
			return 1, "year"
		}
		return timeInYears, "years"
	}

	timeInMonths := round(floatMillis / millisInOneMonth)
	if timeInMonths >= 1 {
		if timeInMonths == 1 {
			return 1, "month"
		}
		return timeInMonths, "months"
	}

	timeInDays := round(floatMillis / millisInOneDay)
	if timeInDays >= 1 {
		if timeInDays == 1 {
			return 1, "day"
		}
		return timeInDays, "days"
	}

	timeInHours := round(floatMillis / millisInOneHour)
	if timeInHours == 1 {
		return 1, "hour"
	}
	return timeInHours, "hours"
}
