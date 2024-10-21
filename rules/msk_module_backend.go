package rules

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// MskModuleBackendRule checks whether an MSK module has an S3 backend defined with the following restrictions:
//   - the key is in the format ${env}-${platform}/${msk-cluster}-${team-name}
//   - the bucket contains the environment in its name
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

func (r *MskModuleBackendRule) getBackendContent(runner tflint.Runner) (*hclext.BodyContent, error) {
	//nolint:wrapcheck
	return runner.GetModuleContent(&hclext.BodySchema{
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
									{Name: "bucket"},
									{Name: "key"},
								},
							},
						},
					},
				},
			},
		},
	}, nil)
}

func (r *MskModuleBackendRule) Check(runner tflint.Runner) error {
	isRoot, err := isRootModule(runner)
	if err != nil {
		return err
	}
	if !isRoot {
		logger.Debug("skipping child module")
		return nil
	}

	content, err := r.getBackendContent(runner)
	if err != nil {
		return fmt.Errorf("getting module content: %w", err)
	}

	backend, err := r.validateBackendDef(runner, content)
	if err != nil {
		return err
	}
	if backend == nil {
		return nil
	}

	modInfo, err := r.parseModuleInfo(runner, backend)
	if err != nil {
		return err
	}
	if modInfo == nil {
		return nil
	}

	if err := r.checkBackendBucketFormat(runner, backend, *modInfo); err != nil {
		return err
	}
	return r.checkBackendKeyFormat(runner, backend, *modInfo)
}

func (r *MskModuleBackendRule) validateBackendDef(
	runner tflint.Runner,
	content *hclext.BodyContent,
) (*hclext.Block, error) {
	backend := findBackendDef(content)
	if backend == nil {
		err := runner.EmitIssue(r, "an s3 backend should be configured for a kafka MSK module", hcl.Range{})
		if err != nil {
			return nil, fmt.Errorf("emitting issue: backend missing: %w", err)
		}
		return nil, nil
	}

	backendType := backend.Labels[0]
	if backendType != "s3" {
		err := runner.EmitIssue(r, "backend should always be s3 for a kafka MSK module", backend.DefRange)
		if err != nil {
			return nil, fmt.Errorf("emitting issue: always s3: %w", err)
		}
		return nil, nil
	}
	return backend, nil
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

type moduleInfo struct {
	env        string
	teamName   string
	mskCluster string
}

func (r *MskModuleBackendRule) checkBackendBucketFormat(
	runner tflint.Runner,
	backend *hclext.Block,
	mi moduleInfo,
) error {
	bucketAttr, bucketExists := backend.Body.Attributes["bucket"]
	if !bucketExists {
		err := runner.EmitIssue(
			r,
			"the s3 backend should specify the bucket inside the kafka MSK module",
			backend.DefRange,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: no s3 bucket: %w", err)
		}
		return nil
	}

	var bucket string
	diags := gohcl.DecodeExpression(bucketAttr.Expr, nil, &bucket)
	if diags.HasErrors() {
		return diags
	}

	diags = gohcl.DecodeExpression(bucketAttr.Expr, nil, &bucket)
	if diags.HasErrors() {
		return diags
	}

	envParts := strings.Split(mi.env, "-")
	if !strings.Contains(bucket, envParts[0]) {
		err := runner.EmitIssue(
			r,
			fmt.Sprintf(
				"backend bucket doesn't contain the env of the module. Current value '%s' should contain env '%s'",
				bucket,
				envParts[0],
			),
			bucketAttr.Range,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: bucket not in the correct format: %w", err)
		}
	}
	return nil
}

func (r *MskModuleBackendRule) checkBackendKeyFormat(runner tflint.Runner, backend *hclext.Block, mi moduleInfo) error {
	keyAttr, keyExists := backend.Body.Attributes["key"]
	if !keyExists {
		err := runner.EmitIssue(
			r,
			"the s3 backend should specify the key inside the kafka MSK module",
			backend.DefRange,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: no s3 key: %w", err)
		}
		return nil
	}

	var key string
	diags := gohcl.DecodeExpression(keyAttr.Expr, nil, &key)
	if diags.HasErrors() {
		return diags
	}

	expectedKey := fmt.Sprintf("%s/%s-%s", mi.env, mi.mskCluster, mi.teamName)

	if key != expectedKey {
		err := runner.EmitIssue(
			r,
			fmt.Sprintf(
				"backend key must have the following format: ${env}-${platform}/${msk-cluster}-${team-name}. Expected: '%s', current: '%s'",
				expectedKey,
				key,
			),
			keyAttr.Range,
		)
		if err != nil {
			return fmt.Errorf("emitting issue: key not in the correct format: %w", err)
		}
	}

	return nil
}

func (r *MskModuleBackendRule) parseModuleInfo(runner tflint.Runner, backend *hclext.Block) (*moduleInfo, error) {
	modulePath, err := runner.GetOriginalwd()
	if err != nil {
		return nil, fmt.Errorf("failed getting module path: %w", err)
	}

	pathElems := strings.Split(filepath.Clean(modulePath), string(filepath.Separator))
	if len(pathElems) < 3 {
		err := runner.EmitIssue(
			r,
			fmt.Sprintf(
				"the module doesn't have the expected structure: the path should end with '${env}-${platform}/${msk-cluster}/${team-name}', but it is: %s",
				modulePath,
			),
			backend.DefRange,
		)
		if err != nil {
			return nil, fmt.Errorf("emitting issue: module not in the right structure: %w", err)
		}
		return nil, nil
	}

	mi := &moduleInfo{
		teamName:   pathElems[len(pathElems)-1],
		mskCluster: pathElems[len(pathElems)-2],
		env:        pathElems[len(pathElems)-3],
	}
	return mi, nil
}
