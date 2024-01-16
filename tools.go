//go:build tools
// +build tools

package tools

import (
	_ "github.com/bakito/semver"
	_ "github.com/norwoodj/helm-docs/cmd/helm-docs"
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "go.uber.org/mock/mockgen"
)
