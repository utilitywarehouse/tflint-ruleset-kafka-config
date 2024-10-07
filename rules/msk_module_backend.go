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
	return ReferenceLink(r.Name())
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

	return r.checkKeyFormat(runner, keyAttr)
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

func (r *MskModuleBackendRule) checkKeyFormat(runner tflint.Runner, keyAttr *hclext.Attribute) error {
	var key string
	diags := gohcl.DecodeExpression(keyAttr.Expr, nil, &key)
	if diags.HasErrors() {
		return diags
	}

	modulePath, err := runner.GetOriginalwd()
	if err != nil {
		return fmt.Errorf("failed getting module path: %w", err)
	}

	pathElems := strings.Split(filepath.Clean(modulePath), string(filepath.Separator))
	if len(pathElems) < 3 {
		return fmt.Errorf("the module doesn't have the expected structure: the path should end with env/msk-cluster/team-name, but it is: %s", modulePath)
	}

	teamName := pathElems[len(pathElems)-1]
	mskCluster := pathElems[len(pathElems)-2]
	env := pathElems[len(pathElems)-3]
	expectedKey := fmt.Sprintf("%s/%s-%s", env, mskCluster, teamName)

	if key != expectedKey {
		err = runner.EmitIssue(
			r,
			fmt.Sprintf("backend key must have the following format: {{env}}/{{cluster}}-{{team-name}}. Expected: '%s', current: '%s'", expectedKey, key),
			keyAttr.Range,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: key not in the correct format: %w", err)
		}
	}

	return nil
}
