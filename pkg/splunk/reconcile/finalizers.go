// Copyright (c) 2018-2020 Splunk Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha2"
)

const (
	splunkFinalizerDeletePVC = "enterprise.splunk.com/delete-pvc"
)

// CheckSplunkDeletion checks to see if deletion was requested for the custom resource.
// If so, it will process and remove any remaining finalizers.
func CheckSplunkDeletion(cr enterprisev1.MetaObject, c ControllerClient) (bool, error) {
	scopedLog := rconcilelog.WithName("CheckSplunkDeletion").WithValues("kind", cr.GetTypeMeta().Kind, "name", cr.GetIdentifier(), "namespace", cr.GetNamespace())
	currentTime := metav1.Now()

	// sanity check: return early if missing GetDeletionTimestamp
	if cr.GetObjectMeta().GetDeletionTimestamp() == nil {
		scopedLog.Info("DeletionTimestamp is nil")
		return false, nil
	}

	// just log warning if deletion time is in the future
	if !cr.GetObjectMeta().GetDeletionTimestamp().Before(&currentTime) {
		scopedLog.Info("DeletionTimestamp is in the future",
			"Now", currentTime,
			"DeletionTimestamp", cr.GetObjectMeta().GetDeletionTimestamp())
		//return false, nil
	}

	scopedLog.Info("Deletion requested")

	// process each finalizer
	for _, finalizer := range cr.GetObjectMeta().GetFinalizers() {
		switch finalizer {
		case splunkFinalizerDeletePVC:
			if err := DeleteSplunkPvc(cr, c); err != nil {
				return false, err
			}
			if err := RemoveSplunkFinalizer(cr, c, finalizer); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("Finalizer in %s %s/%s not recognized: %s", cr.GetTypeMeta().Kind, cr.GetNamespace(), cr.GetIdentifier(), finalizer)
		}
	}

	scopedLog.Info("Deletion complete")

	return true, nil
}

// DeleteSplunkPvc removes all corresponding PersistentVolumeClaims that are associated with a custom resource.
func DeleteSplunkPvc(cr enterprisev1.MetaObject, c ControllerClient) error {
	scopedLog := rconcilelog.WithName("DeleteSplunkPvc").WithValues("kind", cr.GetTypeMeta().Kind, "name", cr.GetIdentifier(), "namespace", cr.GetNamespace())

	var component string
	switch cr.GetTypeMeta().Kind {
	case "Standalone":
		component = "standalone"
	case "LicenseMaster":
		component = "license-master"
	case "SearchHeadCluster":
		component = "search-head"
	case "IndexerCluster":
		component = "indexer"
	default:
		scopedLog.Info("Skipping PVC removal")
		return nil
	}

	// get list of PVCs for this cluster
	labels := map[string]string{
		"app.kubernetes.io/part-of": fmt.Sprintf("splunk-%s-%s", cr.GetIdentifier(), component),
	}
	listOpts := []client.ListOption{
		client.InNamespace(cr.GetNamespace()),
		client.MatchingLabels(labels),
	}
	pvclist := corev1.PersistentVolumeClaimList{}
	if err := c.List(context.Background(), &pvclist, listOpts...); err != nil {
		return err
	}

	if len(pvclist.Items) == 0 {
		scopedLog.Info("No PVC found")
		return nil
	}

	// delete each PVC
	for _, pvc := range pvclist.Items {
		scopedLog.Info("Deleting PVC", "name", pvc.ObjectMeta.Name)
		if err := c.Delete(context.Background(), &pvc); err != nil {
			return err
		}
	}

	return nil
}

// RemoveSplunkFinalizer removes a finalizer from a custom resource.
func RemoveSplunkFinalizer(cr enterprisev1.MetaObject, c ControllerClient, finalizer string) error {
	scopedLog := rconcilelog.WithName("RemoveSplunkFinalizer").WithValues("kind", cr.GetTypeMeta().Kind, "name", cr.GetIdentifier(), "namespace", cr.GetNamespace())
	scopedLog.Info("Removing finalizer", "name", finalizer)

	// create new list of finalizers that doesn't include the one being removed
	var newFinalizers []string

	// handles multiple occurrences (performance is not significant)
	for _, f := range cr.GetObjectMeta().GetFinalizers() {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}

	// update object
	cr.GetObjectMeta().SetFinalizers(newFinalizers)
	return c.Update(context.Background(), cr)
}
