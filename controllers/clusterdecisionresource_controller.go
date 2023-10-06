/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	argogeneratorsv1alpha1 "github.com/bashkimkasa/cdr-operator/api/v1alpha1"
)

// Definitions for configmap to watch
var (
	// Argo name and namespace of configMap that stores cluster configurations
	configMapName      = getenv("ARGO_CONFIGMAP_NAME", "cluster-config")
	configMapNamespace = getenv("ARGO_CONFIGMAP_NAMESPACE", "default")
)

// ClusterDecisionResourceReconciler reconciles a ClusterDecisionResource object
type ClusterDecisionResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ClusterConfigData struct {
	Clusters []map[string]string `yaml:"clusters"`
}

//+kubebuilder:rbac:groups=argo.generators.bashkimkasa,resources=clusterdecisionresources,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=argo.generators.bashkimkasa,resources=clusterdecisionresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

// Reconcile loop
func (r *ClusterDecisionResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Step 0 - Define and setup resource variables
	cr := &argogeneratorsv1alpha1.ClusterDecisionResource{}
	configmap := &corev1.ConfigMap{}

	// Step 1 - Fetch CR instance
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found, ignore and stop the reconciliation
			log.Info("CDR Custom resource not found; ignoring...")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get cdr custom resource")
		return ctrl.Result{}, err
	}

	// Step 2 - Get configmap
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: configMapNamespace}, configmap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the configmap is not found, ignore and stop the reconciliation
			log.Info("Configmap not found; ignoring...")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get configmap")
		return ctrl.Result{}, err
	}

	// Step 3 - Calculate decisions
	decisions, err := calculateDecisions(cr, configmap)
	if err != nil {
		log.Error(err, "Failed to calculate decisions")
		return ctrl.Result{}, err
	}

	//Step 4 - Update CR if decisions have changed
	if cr.Status.Decisions == nil || !reflect.DeepEqual(cr.Status.Decisions, decisions) {
		//Update cr object with new decisions
		cr.Status.Decisions = decisions
		if err = r.Update(ctx, cr); err != nil {
			if apierrors.IsConflict(err) {
				// The CR has been updated since we read it; reqeue the CR to try to reconciliate again.
				return ctrl.Result{Requeue: true}, nil
			}
			if apierrors.IsNotFound(err) {
				// The CR has been deleted since we read it; requeue the CR to try to reconciliate again.
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to update CR status with decisions")
			return ctrl.Result{}, err
		}
		log.Info("CR updated successfully!")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterDecisionResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argogeneratorsv1alpha1.ClusterDecisionResource{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.getRequestsFromConfigMap),
		).
		Complete(r)
}

// Helper functions
// Get environment variable with fallback default
func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

// Calculate the decisions and store in status of the CR
func calculateDecisions(cr *argogeneratorsv1alpha1.ClusterDecisionResource, configmap *corev1.ConfigMap) ([]map[string]string, error) {
	// TODO - implement real logic here - for the time being implementing a fake logic
	desired_clusters := cr.Spec.ClusterList
	max_clusters := cr.Spec.MaxClusters
	decisions := []map[string]string{}
	config_clusters_string := configmap.Data["config.yml"]
	config_clusters := ClusterConfigData{}

	err := yaml.Unmarshal([]byte(config_clusters_string), &config_clusters)
	if err != nil {
		log.Log.Error(err, "Not able to parse configmap data")
		return nil, err
	}

	temp_clusters := []map[string]string{}

	for _, desired_cluster := range desired_clusters {
		for _, cluster := range config_clusters.Clusters {
			clusterName := cluster["clusterName"]
			status := cluster["status"]

			if status == "ACTIVE" && clusterName == desired_cluster {
				temp_clusters = append(temp_clusters, map[string]string{"clusterName": clusterName})
			}
		}
	}

	i := 0
	for _, cluster := range temp_clusters {
		if i < max_clusters {
			decisions = append(decisions, cluster)
		}
		i++
	}

	return decisions, nil
}

func (r *ClusterDecisionResourceReconciler) getRequestsFromConfigMap(configMap client.Object) []reconcile.Request {
	//The config map (with matching name and namespace) changes affects all of the CRs of controller managed type (ClusterDecisionResource) so no need to filter using list options
	if configMap.GetName() != configMapName || configMap.GetNamespace() != configMapNamespace {
		return []reconcile.Request{}
	}

	crs := &argogeneratorsv1alpha1.ClusterDecisionResourceList{}
	err := r.List(context.Background(), crs)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(crs.Items))
	for i, item := range crs.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
