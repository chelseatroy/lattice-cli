package test_helpers

import (
	"fmt"
	"regexp"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/types"
)

func Say(expected string, args ...interface{}) types.GomegaMatcher {
	return gbytes.Say(regexSafe(expected))
}

func SayNewLine() types.GomegaMatcher {
	return Say("\n")
}

var regex = regexp.MustCompile("[-/\\\\^$*+?.()|[\\]{}]")

func regexSafe(matcher string) string {
	return regex.ReplaceAllStringFunc(matcher, func(s string) string {
		return fmt.Sprintf("\\%s", s)
	})
}
