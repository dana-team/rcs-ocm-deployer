package e2e_tests

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	TimeoutCapp              = 60 * time.Second
	CappCreationInterval     = 2 * time.Second
	DefaultEventuallySeconds = 2
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Second * DefaultEventuallySeconds)
	RunSpecs(t, "RCS Suite")
}
