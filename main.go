package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/utilitywarehouse/tflint-ruleset-kafka-config/rules"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "kafka-config",
			Version: "0.1.0",
			Rules: []tflint.Rule{
				rules.NewTerraformBackendTypeRule(),
			},
		},
	})
}
