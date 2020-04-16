package testenv

import (
	"context"
	"fmt"
	"time"
	"flag"

	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	wait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/go-logr/logr"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
)

const (
	defaultOperatorImage	= "splunk/splunk-operator"
	defaultSplunkImage		= "splunk/splunk:latest"
	defaultSparkImage		= "splunk/spark"

	// PollInterval specifies the polling interval 
	PollInterval = 1 * time.Second
	// DefaultTimeout is the max timeout before we failed.
	DefaultTimeout = 5 * time.Minute
)

var (
	metricsHost       	= "0.0.0.0"
	metricsPort int32 	= 8383
	specifiedOperatorImage	= defaultOperatorImage
	specifiedSplunkImage	= defaultSplunkImage
	specifiedSparkImage		= defaultSparkImage
	specifiedSkipTeardown	= false
)

type cleanupFunc func() error

// TestEnv represents a namespaced-isolated k8s cluster environment to run tests against
type TestEnv struct {
	kubeAPIServer		string
	name        		string
	namespace			string
	serviceAccountName	string
	roleName			string
	roleBindingName		string
	operatorName		string
	operatorImage		string
	splunkImage			string
	sparkImage			string
	initialized			bool
	skipTeardown		bool
	kubeClient  		client.Client
	Log					logr.Logger
	cleanupFuncs		[]cleanupFunc
}


func init() {
	l := zap.LoggerTo(ginkgo.GinkgoWriter)
	l.WithName("testenv")
	logf.SetLogger(l) 

	flag.StringVar(&specifiedOperatorImage, "operator", defaultOperatorImage, "operator image to use")
	flag.StringVar(&specifiedSplunkImage, "splunk", defaultSplunkImage, "splunk enterprise (splunkd) image to use")
	flag.StringVar(&specifiedSparkImage, "spark", defaultSparkImage, "spark image to use")
	flag.BoolVar(&specifiedSkipTeardown, "skip-teardown", false, "True to skip tearing down the test env after use")
}

// GetKubeClient returns the kube client to talk to kube-apiserver
func (testenv *TestEnv) GetKubeClient() client.Client {
	return testenv.kubeClient
}

// NewDefaultTestEnv creates a default test environment
func NewDefaultTestEnv(name string) (*TestEnv, error) {
	return NewTestEnv(name, specifiedOperatorImage, specifiedSplunkImage, specifiedSparkImage)
}

// NewTestEnv creates a new test environment to run tests againsts
func NewTestEnv(name, operatorImage, splunkImage, sparkImage string) (*TestEnv, error) {
	
	testenv := &TestEnv{
		name: 				name,
		namespace:			"ns-" + name,
		serviceAccountName: "sa-" + name,
		roleName: 			"role-" + name,
		roleBindingName:	"rolebinding-" + name,
		operatorName:		"op-" + name,
		operatorImage:		operatorImage,
		splunkImage:		splunkImage,
		sparkImage:			sparkImage,
		skipTeardown:		specifiedSkipTeardown,
	}

	testenv.Log = logf.Log.WithValues("testenv", testenv.name)

	// Scheme
	enterprisev1.SchemeBuilder.AddToScheme(scheme.Scheme)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	testenv.kubeAPIServer = cfg.Host
	testenv.Log.Info("Using kube-apiserver\n", "kube-apiserver", cfg.Host)

	// 
	metricsAddr := fmt.Sprintf("%s:%d", metricsHost, metricsPort + int32(ginkgoconfig.GinkgoConfig.ParallelNode))

	kubeManager, err := manager.New(cfg, manager.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: metricsAddr,
	})
	if err != nil {
		return nil, err
	}

	testenv.kubeClient = kubeManager.GetClient()
	if testenv.kubeClient == nil {
		return nil, fmt.Errorf("kubeClient is nil")
	}


	// We need to start the manager to setup the cache. Otherwise, we have to
	// use apireader instead of kubeclient when retrieving resources
	go func() {
		err := kubeManager.Start(signals.SetupSignalHandler())
		if err != nil {
			panic("Unable to start kube manager. Error: " + err.Error())
		}
	}()

	testenv.Log.Info("testenv created.\n")
	return testenv, nil
}

//GetName returns the name of the testenv
func (testenv *TestEnv) GetName() string {
	return testenv.name
}

// Initialize initializes the testenv
func (testenv *TestEnv) Initialize() error {
	testenv.Log.Info("testenv initializing.\n")

	var err error
	err = testenv.createNamespace()
	if err != nil {
		return err
	}

	err = testenv.createSA()
	if err != nil {
		return err
	}

	err = testenv.createRole()
	if err != nil {
		return err
	}

	err = testenv.createRoleBinding()
	if err != nil {
		return err
	}
	
	err = testenv.createOperator()
	if err != nil {
		return err
	}

	testenv.initialized = true
	testenv.Log.Info("testenv initialized.\n", "namespace", testenv.namespace, "operatorImage", testenv.operatorImage, "splunkImage", testenv.splunkImage, "sparkImage", testenv.sparkImage)
	return nil
}

// Destroy destroy the testenv
func (testenv *TestEnv) Destroy() error {

	if testenv.skipTeardown {
		testenv.Log.Info("testenv teardown is skipped!\n")
		return nil
	}

	testenv.initialized = false

	for fn, err := testenv.popCleanupFunc(); err == nil; fn, err = testenv.popCleanupFunc() {
		cleanupErr := fn()
		if cleanupErr != nil {
			testenv.Log.Error(cleanupErr, "CleanupFunc returns an error. Attempt to continue.\n" )
		}
	}

	testenv.Log.Info("testenv deleted.\n")
	return nil
}

// GetStandalone retrieves the standalone object
func (testenv *TestEnv) GetStandalone(name string) (*enterprisev1.Standalone, error) {
	key := client.ObjectKey {Name: name, Namespace: testenv.namespace}

	standalone := &enterprisev1.Standalone{}
	err := testenv.GetKubeClient().Get(context.TODO(), key, standalone)
	if err != nil {
		return nil, err
	}
	return standalone, nil
}


// CreateStandalone creates a standalone deployment
func (testenv *TestEnv) CreateStandalone(name string) (*enterprisev1.Standalone, error) {
	standalone := createStandaloneCR(name, testenv.namespace)
	err := testenv.GetKubeClient().Create(context.TODO(), standalone)
	if err != nil {
		return nil, err
	}

	// Returns once we can retrieve the standalone instance
	if err := wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		key := client.ObjectKey {Name: name, Namespace: testenv.namespace}
		err := testenv.GetKubeClient().Get(context.TODO(), key, standalone)
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

	return standalone, nil
}

// DeleteStandalone deletes the standalone deployment
func (testenv *TestEnv) DeleteStandalone(name string) error {
	standalone := createStandaloneCR(name, testenv.namespace)
	return testenv.GetKubeClient().Delete(context.TODO(), standalone)
}

func (testenv *TestEnv) pushCleanupFunc(fn cleanupFunc) {
	testenv.cleanupFuncs = append(testenv.cleanupFuncs, fn)
}

func (testenv *TestEnv) popCleanupFunc() (cleanupFunc, error) {
	if len(testenv.cleanupFuncs) == 0 {
		return nil, fmt.Errorf("cleanupFuncs is empty")
	}

	fn := testenv.cleanupFuncs[len(testenv.cleanupFuncs)-1]
	testenv.cleanupFuncs = testenv.cleanupFuncs[:len(testenv.cleanupFuncs) -1]

	return fn, nil
}

func (testenv *TestEnv) createNamespace() error {

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testenv.namespace,
		},
	}

	err := testenv.GetKubeClient().Create(context.TODO(), namespace)
	if err != nil {
		return err
	}

	// Cleanup the namespace when we teardown this testenv
	testenv.pushCleanupFunc(func() error {
		testenv.Log.Info("Deleting namespace")
		err := testenv.GetKubeClient().Delete(context.TODO(), namespace)
		if err != nil {
			return err
		}	
		if err = wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
			key := client.ObjectKey {Name: testenv.namespace, Namespace: testenv.namespace}
			ns := &corev1.Namespace{}
			err := testenv.GetKubeClient().Get(context.TODO(), key, ns)
			if errors.IsNotFound(err) {
				return true, nil 
			}
			if ns.Status.Phase == corev1.NamespaceTerminating{
				return false, nil
			}
			
			return true, nil
		}); err != nil {
			return err
		}
	
		return nil
	})

	if err := wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		key := client.ObjectKey {Name: testenv.namespace }
		ns := &corev1.Namespace{}
		err := testenv.GetKubeClient().Get(context.TODO(), key, ns)
		if err != nil {
			// Try again
			if errors.IsNotFound(err) {
				return false, nil 
			}
			return false, err
		}
		if ns.Status.Phase == corev1.NamespaceActive {
			return true, nil
		}
		
		return false, nil
	}); err != nil {
		return err
	}

	return nil
}

func (testenv *TestEnv) createSA() error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:    testenv.serviceAccountName,
			Namespace: testenv.namespace,
		},
	}

	err := testenv.GetKubeClient().Create(context.TODO(), sa)
	if err != nil {
		return err
	}

	testenv.pushCleanupFunc(func() error {
		testenv.Log.Info("Deleting SA")
		err := testenv.GetKubeClient().Delete(context.TODO(), sa)
		if err != nil {
			return err
		}
		return nil		
	})

	return nil
}

func (testenv *TestEnv) createRole() error {
	role := createRole(testenv.roleName, testenv.namespace)

	err := testenv.GetKubeClient().Create(context.TODO(), role)
	if err != nil {
		return err
	}

	testenv.pushCleanupFunc(func() error {
		testenv.Log.Info("Deleting Role")
		err := testenv.GetKubeClient().Delete(context.TODO(), role)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

func (testenv *TestEnv) createRoleBinding() error {
	binding := createRoleBinding(testenv.roleBindingName, testenv.serviceAccountName, testenv.namespace, testenv.roleName)

	err := testenv.GetKubeClient().Create(context.TODO(), binding)
	if err != nil {
		return err
	}

	testenv.pushCleanupFunc(func() error {
		testenv.Log.Info("Deleting RoleBinding")
		err := testenv.GetKubeClient().Delete(context.TODO(), binding)
		if err != nil {
			return err
		}
		return nil
	})

	return nil
}

func (testenv *TestEnv) createOperator() error {
	op := createOperator(testenv.operatorName, testenv.namespace, testenv.serviceAccountName, testenv.operatorImage, testenv.splunkImage, testenv.sparkImage)
	err := testenv.GetKubeClient().Create(context.TODO(), op)
	if err != nil {
		return err
	}

	testenv.pushCleanupFunc(func() error {
		testenv.Log.Info("Deleting Operator")
		err := testenv.GetKubeClient().Delete(context.TODO(), op)
		if err != nil {
			return err
		}
		return nil
	})

	if err := wait.PollImmediate(PollInterval, DefaultTimeout, func() (bool, error) {
		key := client.ObjectKey {Name: testenv.operatorName, Namespace: testenv.namespace }
		deployment := &appsv1.Deployment{}
		err := testenv.GetKubeClient().Get(context.TODO(), key, deployment)
		if err != nil {
			return false, err
		}

		if deployment.Status.UpdatedReplicas < deployment.Status.Replicas {
			return false, nil
		}

		if deployment.Status.ReadyReplicas < *op.Spec.Replicas {
			return false, nil
		}
		
		return true, nil
	}); err != nil {
		return err
	}
	return nil
}