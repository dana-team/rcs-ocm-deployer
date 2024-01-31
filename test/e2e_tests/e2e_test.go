package e2e_tests

import (
	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(time.Second * testconsts.DefaultEventuallySeconds)
	RunSpecs(t, "RCS Suite")
}
