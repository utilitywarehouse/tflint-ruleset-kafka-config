package rules

import "fmt"

func ReferenceLink(name string) string {
	return fmt.Sprintf("https://github.com/utilitywarehouse/tflint-ruleset-kafka-config/blob/main/rules/%s.md", name)
}
