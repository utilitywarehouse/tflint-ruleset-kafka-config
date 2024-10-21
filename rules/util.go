package rules

import (
	"fmt"

	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

func ReferenceLink(name string) string {
	return fmt.Sprintf("https://github.com/utilitywarehouse/tflint-ruleset-kafka-config/blob/main/rules/%s.md", name)
}

// many of our rules want to look at a root module and collect information
// about all its child modules. Hence they only want to run for root modules
// and not be also invoked on each child module.
func isRootModule(runner tflint.Runner) (bool, error) {
	path, err := runner.GetModulePath()
	if err != nil {
		return false, fmt.Errorf("getting module path: %w", err)
	}

	return path.IsRoot(), nil
}
