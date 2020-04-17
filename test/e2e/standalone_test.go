package e2e

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
	"github.com/splunk/splunk-operator/test/testenv"
)

const (
	// PollInterval specifies the polling interval
	PollInterval = 2 * time.Second
	// DefaultTimeout is the max timeout before we failed.
	DefaultTimeout = 5 * time.Minute

	// ConsistentPollInterval is the interval to use to consistently check a state is stable
	ConsistentPollInterval = 20 * time.Millisecond
	// ConsistentDuration is the duration to use to check a state is stable
	ConsistentDuration = 200 * time.Millisecond
)

var _ = Describe("Standalone deployment", func() {

	var deployment *testenv.Deployment
	var standalone *enterprisev1.Standalone

	BeforeEach(func() {
		var err error

		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(5))
		Expect(err).To(Succeed(), "Unable to create deployment")

		standalone, err = deployment.DeployStandalone(deployment.GetName())
		Expect(err).To(Succeed(), "Unable to create standalone instance - "+standalone.ObjectMeta.Name)

		Eventually(func() enterprisev1.ResourcePhase {
			standalone, err = deployment.GetStandalone(standalone.ObjectMeta.Name)
			if err != nil {
				return enterprisev1.PhaseError
			}
			testenvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
			return standalone.Status.Phase
		}, DefaultTimeout, PollInterval).Should(Equal(enterprisev1.PhaseReady))
	})

	AfterEach(func() {
		deployment.Teardown()
	})

	When("it is deployed", func() {
		It("it is ready and stable state", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))

			strver := standalone.ObjectMeta.GetResourceVersion()

			Consistently(func() string {
				instance, _ := deployment.GetStandalone(standalone.ObjectMeta.Name)

				return instance.ObjectMeta.GetResourceVersion()
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(strver))
		})

		It("we can update volumes", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		})

		It("we can update service ports", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		})
	})
})
