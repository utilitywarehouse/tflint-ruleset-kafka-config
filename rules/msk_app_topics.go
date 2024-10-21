package rules

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/terraform-linters/tflint-plugin-sdk/hclext"
	"github.com/terraform-linters/tflint-plugin-sdk/logger"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
	"github.com/zclconf/go-cty/cty"
)

// MSKAppTopicsRule checks whether an MSK module only consumes from topics
// defined in the module.
type MSKAppTopicsRule struct {
	tflint.DefaultRule
}

func (r *MSKAppTopicsRule) Name() string {
	return "msk_app_topics"
}

func (r *MSKAppTopicsRule) Enabled() bool {
	return true
}

func (r *MSKAppTopicsRule) Link() string {
	return ReferenceLink(r.Name())
}

func (r *MSKAppTopicsRule) Severity() tflint.Severity {
	return tflint.ERROR
}

func (r *MSKAppTopicsRule) Check(runner tflint.Runner) error {
	isRoot, err := isRootModule(runner)
	if err != nil {
		return err
	}
	if !isRoot {
		logger.Debug("skipping child module")
		return nil
	}

	// resourceNameMap: resource_name -> topic_name (for mapping variables to EvalCtx)
	// moduleTopics: topic_name -> struct{} (for name lookups)
	resourceNameMap, moduleTopics, err := getKafkaTopics(runner)
	if err != nil {
		return err
	}
	logger.Debug("found topics", "topics", resourceNameMap)

	modules, err := runner.GetModuleContent(
		&hclext.BodySchema{
			Blocks: []hclext.BlockSchema{
				{
					Type:       "module",
					LabelNames: []string{"name"},
					Body: &hclext.BodySchema{
						Attributes: []hclext.AttributeSchema{
							{Name: "produce_topics"},
							{Name: "consume_topics"},
						},
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("getting modules: %w", err)
	}
	evalCtx := buildTopicNameContext(resourceNameMap)
	for _, block := range modules.Blocks {
		for _, topicAttr := range []string{"consume_topics", "produce_topics"} {
			if err := r.reportExternalTopics(runner, topicAttr, block, evalCtx, moduleTopics); err != nil {
				return err
			}
		}
	}
	return nil
}

func getKafkaTopics(runner tflint.Runner) (map[string]string, map[string]struct{}, error) {
	resourceContents, err := runner.GetResourceContent(
		"kafka_topic",
		&hclext.BodySchema{
			Attributes: []hclext.AttributeSchema{{Name: "name"}},
		},
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("getting kafka_topic contents: %w", err)
	}

	resourceNameMap := map[string]string{}
	topicNameMap := map[string]struct{}{}
	for _, topicResource := range resourceContents.Blocks {
		resourceName := topicResource.Labels[1]
		nameAttr := topicResource.Body.Attributes["name"]

		var name string
		diags := gohcl.DecodeExpression(nameAttr.Expr, nil, &name)
		if diags.HasErrors() {
			return nil, nil, fmt.Errorf(
				"decoding name for kafka_topic '%s': %w",
				resourceName,
				diags,
			)
		}
		resourceNameMap[resourceName] = name
		topicNameMap[name] = struct{}{}
	}

	return resourceNameMap, topicNameMap, nil
}

func buildTopicNameContext(topicNameMap map[string]string) *hcl.EvalContext {
	// tflint doesn't do any variable expansion, so we manually build an
	// EvalContext that we can use for lookups of variables like
	// `kafka_topic.my_topic.name` via a lookup like:
	// EvalContext.Variables["kafka_topic"]["my_topic"]["name"]
	nameMap := map[string]cty.Value{}
	for topicResourceName, topicName := range topicNameMap {
		nameMap[topicResourceName] = cty.ObjectVal(
			map[string]cty.Value{"name": cty.StringVal(topicName)},
		)
	}

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"kafka_topic": cty.ObjectVal(nameMap),
		},
	}
}

func (r *MSKAppTopicsRule) reportExternalTopics(
	runner tflint.Runner,
	attrName string,
	block *hclext.Block,
	evalCtx *hcl.EvalContext,
	moduleTopicNames map[string]struct{},
) error {
	topicAttr, ok := block.Body.Attributes[attrName]
	if !ok {
		logger.Debug("skipping block, doesn't provide producer/consumer", "labels", block.Labels)
		return nil
	}

	val, diags := topicAttr.Expr.Value(evalCtx)
	if diags.HasErrors() {
		return fmt.Errorf("evaluating topic names: %w", diags)
	}
	for _, v := range val.AsValueSlice() {
		name := v.AsString()
		if _, ok := moduleTopicNames[name]; !ok {
			err := runner.EmitIssue(
				r,
				fmt.Sprintf(
					"'%s' may only contain topics defined in the current module but '%s' is not",
					attrName,
					name,
				),
				topicAttr.Range,
			)
			if err != nil {
				return fmt.Errorf("emitting issue: %w", err)
			}
		}
	}

	return nil
}
