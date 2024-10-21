package rules

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

const commonNameAttribute = "cert_common_name"

type MSKUniqueAppNamesRule struct {
	tflint.DefaultRule
}

func (r *MSKUniqueAppNamesRule) Name() string {
	return "msk_unique_app_names"
}

func (r *MSKUniqueAppNamesRule) Enabled() bool {
	return true
}

func (r *MSKUniqueAppNamesRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKUniqueAppNamesRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKUniqueAppNamesRule) Check(runner tflint.Runner) error {
	isRoot, err := isRootModule(runner)
	if err != nil {
		return err
	}
	if !isRoot {
		logger.Debug("skipping child module")
		return nil
	}

	TLSAppModules, err := getTLSAppModules(runner)
	if err != nil {
		return err
	}

	return r.reportDuplicateTLSAppNames(runner, TLSAppModules)
}

func getTLSAppModules(runner tflint.Runner) (hclext.Blocks, error) {
	modules, err := runner.GetModuleContent(
		&hclext.BodySchema{
			Blocks: []hclext.BlockSchema{
				{
					Type:       "module",
					LabelNames: []string{"name"},
					Body: &hclext.BodySchema{
						Attributes: []hclext.AttributeSchema{
							{Name: commonNameAttribute},
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

	var TLSAppModules hclext.Blocks
	for _, moduleBlock := range modules.Blocks {
		if _, ok := moduleBlock.Body.Attributes[commonNameAttribute]; ok {
			TLSAppModules = append(TLSAppModules, moduleBlock)
		}
	}

	return TLSAppModules, nil
}

type tlsAppName struct {
	attr *hclext.Attribute
	name string
}

func (r *MSKUniqueAppNamesRule) reportDuplicateTLSAppNames(runner tflint.Runner, tlsAppModules hclext.Blocks) error {
	seenNames := map[string]struct{}{}
	duplicateNames := []tlsAppName{}
	for _, appModule := range tlsAppModules {
		appNameAttr := appModule.Body.Attributes[commonNameAttribute]

		var appName string
		diags := gohcl.DecodeExpression(appNameAttr.Expr, nil, &appName)
		if diags.HasErrors() {
			return fmt.Errorf("decoding expression for attribute %s: %w", commonNameAttribute, diags)
		}

		if _, ok := seenNames[appName]; ok {
			duplicateNames = append(duplicateNames, tlsAppName{attr: appNameAttr, name: appName})
			continue
		}

		seenNames[appName] = struct{}{}
	}

	for _, appName := range duplicateNames {
		if err := runner.EmitIssue(
			r,
			fmt.Sprintf(
				"'%s' must be unique across a module, but '%s' has already been seen",
				commonNameAttribute,
				appName.name,
			),
			appName.attr.Range,
		); err != nil {
			return fmt.Errorf("emitting issue: %w", err)
		}
	}

	return nil
}
