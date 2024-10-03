package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MskModuleBackendRule checks whether an MSK module has an S3 backend defined with a key that has as suffix the name of the team.
type MskModuleBackendRule struct {
	tflint.DefaultRule
}

// NewMskModuleBackendRule returns a new rule.
func NewMskModuleBackendRule() *MskModuleBackendRule {
	return &MskModuleBackendRule{}
}

// Name returns the rule name.
func (r *MskModuleBackendRule) Name() string {
	return "msk_module_backend"
}

// Enabled returns whether the rule is enabled by default.
func (r *MskModuleBackendRule) Enabled() bool {
	return true
}

// Severity returns the rule severity.
func (r *MskModuleBackendRule) Severity() tflint.Severity {
	return tflint.ERROR
}

// Link returns the rule reference link.
func (r *MskModuleBackendRule) Link() string {
	return ""
}

func (r *MskModuleBackendRule) Check(runner tflint.Runner) error {
	path, err := runner.GetModulePath()
	if err != nil {
		return fmt.Errorf("getting module path: %w", err)
	}
	if !path.IsRoot() {
		// This rule does not evaluate child modules.
		return nil
	}

	// This rule is an example to get attributes of blocks other than resources.
	content, err := runner.GetModuleContent(&hclext.BodySchema{
		Blocks: []hclext.BlockSchema{
			{
				Type: "terraform",
				Body: &hclext.BodySchema{
					Blocks: []hclext.BlockSchema{
						{
							Type:       "backend",
							LabelNames: []string{"type"},
							Body: &hclext.BodySchema{
								Attributes: []hclext.AttributeSchema{
									{Name: "key"},
								},
							},
						},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("getting module content: %w", err)
	}

	backend := findBackendDef(content)
	if backend == nil {
		err := runner.EmitIssue(r, "an s3 backend should be configured for a kafka MSK module", hcl.Range{})
		if err != nil {
			return fmt.Errorf("emitting issue: backend missing: %w", err)
		}
		return nil
	}

	backendType := backend.Labels[0]
	if backendType != "s3" {
		err := runner.EmitIssue(r, "backend should always be s3 for a kafka MSK module", backend.DefRange)
		if err != nil {
			return fmt.Errorf("emitting issue: always s3: %w", err)
		}
		return nil
	}

	keyAttr, keyExists := backend.Body.Attributes["key"]
	if !keyExists {
		err := runner.EmitIssue(r, "the s3 backend should specify the details inside the kafka MSK module", backend.DefRange)
		if err != nil {
			return fmt.Errorf("emitting issue: no s3 details: %w", err)
		}
		return nil
	}

	return r.checkKeyHasTeamSuffix(runner, keyAttr)
}

func findBackendDef(content *hclext.BodyContent) *hclext.Block {
	if content.IsEmpty() {
		return nil
	}
	for _, tfConfig := range content.Blocks {
		if len(tfConfig.Body.Blocks) > 0 {
			return tfConfig.Body.Blocks[0]
		}
	}
	return nil
}

func (r *MskModuleBackendRule) checkKeyHasTeamSuffix(runner tflint.Runner, keyAttr *hclext.Attribute) error {
	var key string
	diags := gohcl.DecodeExpression(keyAttr.Expr, nil, &key)
	if diags.HasErrors() {
		return diags
	}

	modulePath, err := runner.GetOriginalwd()
	if err != nil {
		return fmt.Errorf("failed getting module path: %w", err)
	}

	teamName := filepath.Base(modulePath)

	if !strings.HasSuffix(key, teamName) {
		err = runner.EmitIssue(
			r,
			fmt.Sprintf("backend key must have the team's name '%s' as a suffix. Current value is: %s", teamName, key),
			keyAttr.Range,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: no team suffix: %w", err)
		}
	}

	return nil
}
