# Integration and e2e testing for Splunk Operator

## Overview

This README describes how to run the various tests and how to write additional tests

## Running the tests

To run the test, you need the following requirements

### Test Requirements

1. Installed Ginkgo/Gomega package (<https://onsi.github.io/ginkgo/)>
1. Access to existing kubernetes test cluster
1. Installed the enterprise CRDs (kubectl apply -f deploy/crds)

### Run all the tests

1. cd ./test folder
2. ginkgo -v -r -progress

### Run specific test

1. cd ./test/{specific-test} folder
2. ginkgo -v -progress

### Run test using specific operator and splunk images (HIGHLY RECOMMENDED)

1. cd ./test folder
2. ginkgo -v -progress --operator-image=splunk/splunk-operator:test --splunk-image=localhost:5000/splunk:latest

## Writing a test

To write a test, it is helpful to understand the test framework, basic test models and conventions.

### Test framework, basic models and conventions

We are using Ginkgo/Gomega test framework. Ginkgo test framework has test suite and each
suite contains 1 or more test specs (test cases if you will). For simplicity, a test suite is a simple a folder or package.
Essentially each folder has 1 test suite file and 1 or more test spec files. We added a couple of test models or constructs to
the framework to help with test setup and code-sharing.  

First we have the TestEnv model. The TestEnv represents a kubernetes namespace environment. There is a 1-1 relation between test
suite and TestEnv. When you bring up a test suite, it setup a TestEnv environment. Since all tests (specs) run in a suite, this
means all tests runs within or scope to a TestEnv environment. When the test suite completes, the TestEnv environment is tear down and all k8s
resources are cleanup. Specifically, the following k8s resources are created per TestEnv

1. Namespace
1. Service Account
1. Role
1. RoleBinding
1. Operator/Deployment

Second, we have the Deployment model (No relation to k8s deployment object). This Deployment model is simply used to encapsulate
what we are deploying to the testenv for testing and help with cleanup. We can deploy standalone, clustered, indexers, search heads,
...etc. It does not have to encapsulate a single type of deployment. For example, we can deploy indexers and search heads as a single
deployment to test. Each spec runs against the deployment model. When the test spec completes the deployment is tear down.

### Add a new test spec

1. To add a new test spec, you can either add a new "It" spec in the existing test spec file or add a new test spec file. If you are adding a new test spec file,
it is best to copy an existing test spec file in that suite and modified it.

### Add a new test suite

1. If you are adding a new test suite (ie you want to run the test in a separate TestEnv or k8s namespace), it is best to copy the example folder entirely and modified it

## Notes

1. As mentioned above, by default, we create a TestEnv (aka k8s namespace) for every test suite AND a Deployment for every test spec. It may be time consuming to tear down
the deployment for each test spec especially if we are deploying a large cluster to test. It is possible then to actually create the TestEnv and Deployment at the test suite
level.
