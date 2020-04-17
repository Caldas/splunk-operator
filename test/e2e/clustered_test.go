package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
	"github.com/splunk/splunk-operator/test/testenv"
)

var _ = XDescribe("Clustered deployment", func() {

	var deployment *testenv.Deployment

	BeforeEach(func() {
		var err error

		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(5))
		Expect(err).To(Succeed(), "Unable to create deployment")

		err = deployment.DeployCluster(deployment.GetName())
		Expect(err).To(Succeed(), "Unable to deploy cluster")
	})

	AfterEach(func() {
		deployment.Teardown()
	})

	When("it is deployed", func() {
		It("it is ready and stable state", func() {
			indexerCluster := &enterprisev1.IndexerCluster{}
			err := deployment.GetInstance(deployment.GetName(), indexerCluster)
			Expect(err).To(Succeed(), "Unable to deploy cluster")
		})
	})
})
