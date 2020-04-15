package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/ginkgo/reporters"

	"github.com/splunk/splunk-operator/test/testenv"
)

var (
	//TestEnvInstance represents the global testenv instance used in this test suite
	TestEnvInstance *testenv.TestEnv
)

// TestE2e is the main entry point
func TestE2e(t *testing.T) {

	RegisterFailHandler(Fail)

	junitReporter := reporters.NewJUnitReporter("e2e_junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "E2E Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {

	By("Setting up the test environment")

	testName := fmt.Sprintf("e2e-%s", testenv.RandomDNSName(6))

	var err error
	TestEnvInstance, err = testenv.NewDefaultTestEnv(testName)
	Expect(err).ToNot(HaveOccurred())

	Expect(TestEnvInstance.Initialize()).ToNot(HaveOccurred())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	Expect(TestEnvInstance.Destroy()).ToNot(HaveOccurred())

})
