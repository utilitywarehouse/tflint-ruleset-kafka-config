package rules

import (
	"fmt"
	"strings"

	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

const (
	consumeGroupAttrName = "consume_groups"
	consumeGroupSepChar  = "."
)

type MSKAppConsumeGroupsRule struct {
	tflint.DefaultRule
}

func (r *MSKAppConsumeGroupsRule) Name() string {
	return "msk_app_consume_groups"
}

func (r *MSKAppConsumeGroupsRule) Enabled() bool {
	return true
}

func (r *MSKAppConsumeGroupsRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKAppConsumeGroupsRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKAppConsumeGroupsRule) Check(runner tflint.Runner) error {
	isRoot, err := isRootModule(runner)
	if err != nil {
		return err
	}
	if !isRoot {
		logger.Debug("skipping child module")
		return nil
	}

	appBlocks, err := getTLSApps(runner)
	if err != nil {
		return err
	}

	return r.validateConsumeGroups(runner, appBlocks)
}

func getTLSApps(runner tflint.Runner) (hclext.Blocks, error) {
	modules, err := runner.GetModuleContent(
		&hclext.BodySchema{
			Blocks: []hclext.BlockSchema{
				{
					Type:       "module",
					LabelNames: []string{"name"},
					Body: &hclext.BodySchema{
						Attributes: []hclext.AttributeSchema{
							{Name: consumeGroupAttrName},
						},
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("getting modules: %w", err)
	}

	var appBlocks hclext.Blocks
	for _, block := range modules.Blocks {
		_, ok := block.Body.Attributes[consumeGroupAttrName]
		if !ok {
			logger.Debug("skipping block, doesn't have 'consume_group' attribute", "labels", block.Labels)
			continue
		}
		appBlocks = append(appBlocks, block)
	}

	return appBlocks, nil
}

func (r *MSKAppConsumeGroupsRule) validateConsumeGroups(runner tflint.Runner, appBlocks hclext.Blocks) error {
	for _, block := range appBlocks {
		consumeGroupAttr := block.Body.Attributes[consumeGroupAttrName]

		var consumeGroupNames []string
		if err := runner.EvaluateExpr(consumeGroupAttr.Expr, &consumeGroupNames, nil); err != nil {
			return fmt.Errorf("decoding attribute '%s': %v", consumeGroupAttrName, err)
		}
		for _, name := range consumeGroupNames {
			if !strings.Contains(name, consumeGroupSepChar) {
				err := runner.EmitIssue(
					r,
					fmt.Sprintf(
						"'%s' must be prefixed with the name of the team using it, but '%s' is not",
						consumeGroupAttrName,
						name,
					),
					consumeGroupAttr.Range,
				)
				if err != nil {
					return fmt.Errorf("emitting issue: %w", err)
				}
			}
		}
	}

	return nil
}
