package e2e

import (
	"fmt"
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
	DefaultTimeout = 10 * time.Minute
)


var _ = Describe("Deploy enteprise", func() {

	standaloneName := fmt.Sprintf("standalone-%s", testenv.RandomDNSName(6))
	It ("Deploys standalone instance", func() {
	
		_, err := TestEnvInstance.CreateStandalone(standaloneName)
		Expect(err).To(Succeed(), "Unable to create standalone instance")

		Eventually(func() enterprisev1.ResourcePhase {

			latest, err := TestEnvInstance.GetStandalone(standaloneName)
			if err != nil {
				return enterprisev1.PhaseError
			}
			TestEnvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standaloneName, "Phase", latest.Status.Phase)
			return latest.Status.Phase 
		}, DefaultTimeout, PollInterval).Should(Equal(enterprisev1.PhaseReady))

		//TODO: Add additional expectations eg service is correct, pod status is good...etc
	})
})
