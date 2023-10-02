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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterDecisionResourceSpec defines the desired state of ClusterDecisionResource
type ClusterDecisionResourceSpec struct {
	// Availability choice, maximum number of clusters to provision to
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	MaxClusters int `json:"maxClusters,omitempty"`

	// Ordered list of clusters to provision to
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ClusterList []string `json:"clusterList,omitempty"`
}

// ClusterDecisionResourceStatus defines the Status of ClusterDecisionResource
type ClusterDecisionResourceStatus struct {
	// This is used by the ARGO ClusterDecisionResource generator to get the list of clusters to deploy to
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Decisions []map[string]string `json:"decisions,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterDecisionResource is the Schema for the clusterdecisionresources API
type ClusterDecisionResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDecisionResourceSpec   `json:"spec,omitempty"`
	Status ClusterDecisionResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterDecisionResourceList contains a list of ClusterDecisionResource
type ClusterDecisionResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDecisionResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterDecisionResource{}, &ClusterDecisionResourceList{})
}
