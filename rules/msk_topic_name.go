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

type mskTopicNameRuleConfig struct {
	TeamAliases map[string][]string `hclext:"team_aliases,optional"`
}

// MskTopicNameRule checks whether a topic defined in MSK has an allowed team prefix.
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

	var config mskTopicNameRuleConfig
	err = runner.DecodeRuleConfig(r.Name(), &config)
	if err != nil {
		return fmt.Errorf("decoding rule config: %w", err)
	}

	logger.Debug("decoded rule config: %v", config)

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
		if err := r.validateTopicName(runner, topicResource, teamName, config.TeamAliases); err != nil {
			return err
		}
	}

	return nil
}

func (r *MskTopicNameRule) validateTopicName(
	runner tflint.Runner,
	topic *hclext.Block,
	teamName string,
	aliases map[string][]string,
) error {
	resourceName := topic.Labels[1]
	nameAttr := topic.Body.Attributes["name"]

	var topicName string
	diags := gohcl.DecodeExpression(nameAttr.Expr, nil, &topicName)
	if diags.HasErrors() {
		return fmt.Errorf("decoding name for kafka_topic '%s': %w", resourceName, diags)
	}

	teamAliases := aliases[teamName]
	if hasTeamNameOrAliasPrefix(topicName, teamName, teamAliases) {
		return nil
	}

	var im string
	if len(teamAliases) != 0 {
		im = fmt.Sprintf(
			"topic name must be prefixed with the team name '%s' or one of its aliases '%s'. Current value is '%s'",
			teamName,
			strings.Join(teamAliases, ", "),
			topicName,
		)
	} else {
		im = fmt.Sprintf("topic name must be prefixed with the team name '%s'. Current value is '%s'", teamName, topicName)
	}

	err := runner.EmitIssue(r, im, nameAttr.Range)
	if err != nil {
		return fmt.Errorf("emitting issue: topic name doesn't have the expected prefix: %w", err)
	}
	return nil
}

func hasTeamNameOrAliasPrefix(topicName string, teamName string, aliases []string) bool {
	aliases = append(aliases, teamName)
	for _, value := range aliases {
		if strings.HasPrefix(topicName, value+".") {
			return true
		}
	}
	return false
}
