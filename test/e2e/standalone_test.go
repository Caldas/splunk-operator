package e2e

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
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
	ConsistentPollInterval = 20*time.Millisecond
	// ConsistentDuration is the duration to use to check a state is stable
	ConsistentDuration = 200 * time.Millisecond
)


var _ = Describe("Standalone deployment", func() {

	var standalone *enterprisev1.Standalone

	BeforeEach(func() {
		var err error
		standaloneName := fmt.Sprintf("standalone-%s", testenv.RandomDNSName(6))
		standalone, err = TestEnvInstance.CreateStandalone(standaloneName)

		Expect(err).To(Succeed(), "Unable to create standalone instance - " + standaloneName)

		Eventually(func() enterprisev1.ResourcePhase {
			standalone, err = TestEnvInstance.GetStandalone(standaloneName)
			if err != nil {
				return enterprisev1.PhaseError
			}
			TestEnvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standaloneName, "Phase", standalone.Status.Phase)
			return standalone.Status.Phase 
		}, DefaultTimeout, PollInterval).Should(Equal(enterprisev1.PhaseReady))
	})

	AfterEach(func() {

		if standalone == nil {
			return
		}

		Expect(TestEnvInstance.DeleteStandalone(standalone.ObjectMeta.Name)).To(Succeed(), "Unable to delete standalone instance - " + standalone.ObjectMeta.Name)
		Eventually(func() bool {
			_, err := TestEnvInstance.GetStandalone(standalone.ObjectMeta.Name)
			if errors.IsNotFound(err) {
				return true
			}
			
			TestEnvInstance.Log.Info("Waiting for standalone instance to be deleted", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
			return false
		}, DefaultTimeout, PollInterval).Should(BeTrue())
	})
	
	When("it is deployed", func() {
		It ("it is ready and stable state", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))

			strver := standalone.ObjectMeta.GetResourceVersion()

			Consistently(func() string {
				instance,_ := TestEnvInstance.GetStandalone(standalone.ObjectMeta.Name)

				return instance.ObjectMeta.GetResourceVersion()
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(strver))
		})
		
		XIt ("we can update volumes", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		})
	
		XIt ("we can update service ports", func() {
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
			Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		})
	})
})
