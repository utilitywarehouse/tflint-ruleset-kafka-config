package main

import (
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"

	"github.com/utilitywarehouse/tflint-ruleset-kafka-config/rules"
)

// set by goreleaser at build time: https://goreleaser.com/cookbooks/using-main.version/
var version = "dev"

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "uw-kafka-config",
			Version: version,
			Rules: []tflint.Rule{
				&rules.MSKModuleBackendRule{},
				&rules.MSKAppTopicsRule{},
				&rules.MSKTopicNameRule{},
				&rules.MSKTopicConfigRule{},
				// keep the comments rule after the config one, as the config one might remove some properties checked by the comments one
				&rules.MSKTopicConfigCommentsRule{},
				&rules.MSKUniqueAppNamesRule{},
			},
		},
	})
}
