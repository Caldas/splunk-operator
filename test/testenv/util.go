package testenv

import (
	"time"
	"math/rand"

	rbacv1 "k8s.io/api/rbac/v1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func init() {
    rand.Seed(time.Now().UnixNano())
}

// RandomDNSName returns a random string that is a valid DNA name
func RandomDNSName(n int) string {
    b := make([]byte, n)
    for i := range b {
		// Must start with letter
		if (i == 0) {
  	    	b[i] = letterBytes[rand.Intn(25)]
		}else {
			b[i] = letterBytes[rand.Intn(len(letterBytes))]
		}
    }
    return string(b)
}


// createStandaloneCR creates and initializes CR for Standalone Kind
func createStandaloneCR(name, ns string) *enterprisev1.Standalone {

	new := enterprisev1.Standalone{
		TypeMeta: metav1.TypeMeta{
			Kind: "Standalone",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns,
			Finalizers: []string{"enterprise.splunk.com/delete-pvc"},
		},

		Spec: enterprisev1.StandaloneSpec{
			CommonSplunkSpec: enterprisev1.CommonSplunkSpec{
				Volumes: []corev1.Volume{},
				CommonSpec: enterprisev1.CommonSpec{
					ImagePullPolicy: "IfNotPresent",
				},
			},
		},
	}

	return &new
}

// createIndexerClusterCR creates and initialize the CR for IndexerCluster Kind
func createIndexerClusterCR(name, ns, licenseName string, replicas int) *enterprisev1.IndexerCluster {

	new := enterprisev1.IndexerCluster{
		TypeMeta: metav1.TypeMeta{
			Kind: "IndexerCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  ns,
			Finalizers: []string{"enterprise.splunk.com/delete-pvc"},
		},

		Spec: enterprisev1.IndexerClusterSpec{
			CommonSplunkSpec: enterprisev1.CommonSplunkSpec{
				Volumes: []corev1.Volume{},
				CommonSpec: enterprisev1.CommonSpec{
					ImagePullPolicy: "IfNotPresent",
				},
				LicenseMasterRef: corev1.ObjectReference{
					Name: licenseName,
				},
			},
			Replicas: int32(replicas),
		},
	}

	return &new
}

func createRole(name, ns string) *rbacv1.Role{
	new := rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:    	name,
			Namespace:  ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "persistentvolumeclaims", "configmaps", "secrets", "pods"},
				Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "damonsets", "replicasets", "statefulsets"},
				Verbs: []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{"enterprise.splunk.com"},
				Resources: []string{"*"},
				Verbs: []string{"*"},
			},
		},
	}

	return &new
}

func createRoleBinding(name, subject, ns, role string) *rbacv1.RoleBinding {
	binding := rbacv1.RoleBinding {
		ObjectMeta: metav1.ObjectMeta{
			Name:    name,
			Namespace: ns,
		},
		Subjects: [] rbacv1.Subject {
			{
				Kind: 		"ServiceAccount",
				Name: 		subject,
				Namespace:	ns,
			},
		},

		RoleRef: rbacv1.RoleRef {
			Kind: "Role",
			Name: role,
			APIGroup: "rbac.authorization.k8s.io", 
		},
	}
	
	return &binding
}

func createOperator(name,ns, account, operatorImageAndTag, splunkEnterpriseImageAndTag, sparkImageAndTag string) *appsv1.Deployment {
	var replicas int32 = 1
	
	operator := appsv1.Deployment {
		ObjectMeta: metav1.ObjectMeta{
			Name:    name,
			Namespace: ns,
		},

		Spec: appsv1.DeploymentSpec {
			Replicas: &replicas,
			Selector: &metav1.LabelSelector {
				MatchLabels: map[string]string {
					"name":"splunk-operator",
				},
			}, 
			Template: corev1.PodTemplateSpec {
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string] string {
						"name":"splunk-operator",
					},
				},
				Spec: corev1.PodSpec {
					ServiceAccountName: account,
					Containers: []corev1.Container {
						{
							Name: name,
							Image: operatorImageAndTag,
							ImagePullPolicy: "IfNotPresent",
							Env: []corev1.EnvVar {
								{
									Name: "WATCH_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource {
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								}, {
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource {
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								}, {
									Name:"OPERATOR_NAME",
									Value:"splunk-operator",
								}, {
									Name:"RELATED_IMAGE_SPLUNK_ENTERPRISE",
									Value: splunkEnterpriseImageAndTag,
								}, {
									Name:"RELATED_IMAGE_SPLUNK_SPARK",
									Value: sparkImageAndTag,
								},
							},
						},
					},
				},
			},
		}, 
	}

	return &operator
}
