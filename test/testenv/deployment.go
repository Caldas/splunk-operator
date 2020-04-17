package testenv

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	wait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
)

// Deployment simply represents the deployment (standalone, clustered,...etc) we create on the testenv
type Deployment struct {
	name         string
	testenv      *TestEnv
	cleanupFuncs []cleanupFunc
}

// GetName returns this deployment name
func (d *Deployment) GetName() string {
	return d.name
}

func (d *Deployment) pushCleanupFunc(fn cleanupFunc) {
	d.cleanupFuncs = append(d.cleanupFuncs, fn)
}

func (d *Deployment) popCleanupFunc() (cleanupFunc, error) {
	if len(d.cleanupFuncs) == 0 {
		return nil, fmt.Errorf("cleanupFuncs is empty")
	}

	fn := d.cleanupFuncs[len(d.cleanupFuncs)-1]
	d.cleanupFuncs = d.cleanupFuncs[:len(d.cleanupFuncs)-1]

	return fn, nil
}

// Teardown teardowns the deployment resources
func (d *Deployment) Teardown() error {
	if d.testenv.skipTeardown {
		d.testenv.Log.Info("deployment teardown is skipped!\n")
		return nil
	}

	var cleanupErr error

	for fn, err := d.popCleanupFunc(); err == nil; fn, err = d.popCleanupFunc() {
		cleanupErr = fn()
		if cleanupErr != nil {
			d.testenv.Log.Error(cleanupErr, "Deployment cleanupFunc returns an error. Attempt to continue.\n")
		}
	}

	d.testenv.Log.Info("deployment deleted.\n", "name", d.name)
	return cleanupErr
}

// DeployStandalone deploys a standalone splunk enterprise instance on the specified testenv
func (d *Deployment) DeployStandalone(name string) (*enterprisev1.Standalone, error) {
	standalone := newStandalone(name, d.testenv.namespace)
	deployed, err := d.deployCR(name, standalone)
	if err != nil {
		return nil, err
	}
	return deployed.(*enterprisev1.Standalone), err
}

// GetStandalone retrieves the standalone object
func (d *Deployment) GetStandalone(name string) (*enterprisev1.Standalone, error) {
	key := client.ObjectKey{Name: name, Namespace: d.testenv.namespace}

	standalone := &enterprisev1.Standalone{}
	err := d.testenv.GetKubeClient().Get(context.TODO(), key, standalone)
	if err != nil {
		return nil, err
	}
	return standalone, nil
}

//DeployLicenseMaster deploys the license master instance
func (d *Deployment) DeployLicenseMaster(name string) (*enterprisev1.LicenseMaster, error) {

	if d.testenv.licenseFilePath == "" {
		return nil, fmt.Errorf("No license file path specified")
	}

	lm := newLicenseMaster(name, d.testenv.namespace, d.testenv.licenseCMName)
	deployed, err := d.deployCR(name, lm)
	if err != nil {
		return nil, err
	}
	return deployed.(*enterprisev1.LicenseMaster), err
}

//DeployIndexerCluster deploys the indexer cluster
func (d *Deployment) DeployIndexerCluster(name, licenseMasterName string, count int) (*enterprisev1.IndexerCluster, error) {
	indexer := newIndexerCluster(name, d.testenv.namespace, licenseMasterName, count)
	deployed, err := d.deployCR(name, indexer)
	if err != nil {
		return nil, err
	}
	return deployed.(*enterprisev1.IndexerCluster), err
}

// DeploySearchHeadCluster deploys a search head cluster
func (d* Deployment) DeploySearchHeadCluster(name, indexerClusterName, licenseMasterName string) (*enterprisev1.SearchHeadCluster, error) {
	indexer := newSearchHeadCluster(name, d.testenv.namespace, indexerClusterName, licenseMasterName)
	deployed, err := d.deployCR(name, indexer)
	return deployed.(*enterprisev1.SearchHeadCluster), err
}

func (d *Deployment) deployCR(name string, cr runtime.Object) (runtime.Object, error) {

	err := d.testenv.GetKubeClient().Create(context.TODO(), cr)
	if err != nil {
		return nil, err
	}

	// Push the clean up func to delete the cr when done 
	d.pushCleanupFunc(func() error {
		d.testenv.Log.Info("Deleting cr", "name", name)
		err := d.testenv.GetKubeClient().Delete(context.TODO(), cr)
		if err != nil {
			return err
		}
		if err = wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
			key := client.ObjectKey{Name: name, Namespace: d.testenv.namespace}
			err := d.testenv.GetKubeClient().Get(context.TODO(), key, cr)

			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}); err != nil {
			return err
		}

		return nil
	})

	// Returns once we can retrieve the lm instance
	if err := wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		key := client.ObjectKey{Name: name, Namespace: d.testenv.namespace}
		err := d.testenv.GetKubeClient().Get(context.TODO(), key, cr)
		if err != nil {

			// Try again
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	}); err != nil {
		return nil, err
	}

	return cr, nil
}