/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SshCommandAuditSpec defines the desired state of SshCommandAudit.
type SshCommandAuditSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// SshCommandEntity is an Array  of SshCommandAudit.
	SshCommandEntity []SshCommandEntity `json:"sshCommandEntity,omitempty"`
}

type SshCommandEntity struct {
	Node      string `json:"node,omitempty"`
	TimeStamp string `json:"time,omitempty"`
	User      string `json:"user,omitempty"`
	SrcIp     string `json:"ip,omitempty"`
	SrcPort   string `json:"port,omitempty"`
	WorkDir   string `json:"pwd,omitempty"`
	Command   string `json:"command,omitempty"`
	ExitCode  string `json:"exit_code,omitempty"`
}

// SshCommandAuditStatus defines the observed state of SshCommandAudit.
type SshCommandAuditStatus struct {
	TotalCommands *int32 `json:"totalCommands"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SshCommandAudit is the Schema for the sshcommandaudits API.
type SshCommandAudit struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SshCommandAuditSpec   `json:"spec,omitempty"`
	Status SshCommandAuditStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SshCommandAuditList contains a list of SshCommandAudit.
type SshCommandAuditList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SshCommandAudit `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SshCommandAudit{}, &SshCommandAuditList{})
}
