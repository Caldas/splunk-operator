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
)


var _ = Describe("Deploys standalone enterprise", func() {

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
	
	It ("can deploy", func() {

		Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		//TODO: Add additional expectations eg service is correct, pod status is good...etc
	})


	It ("can update volumes", func() {

		Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		//TODO: Add additional expectations eg service is correct, pod status is good...etc
	})

	It ("can update service", func() {

		Expect(standalone.Status.Phase).To(Equal(enterprisev1.PhaseReady))
		//TODO: Add additional expectations eg service is correct, pod status is good...etc
	})
})
