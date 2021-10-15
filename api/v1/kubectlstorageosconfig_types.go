/*
Copyright 2021.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubectlStorageOSConfigSpec defines the desired state of KubectlStorageOSConfig
type KubectlStorageOSConfigSpec struct {
	StackTrace                bool `json:"stackTrace,omitempty"`
	SkipNamespaceDeletion     bool `json:"skipNamespaceDeletion,omitempty"`
	SkipExistingWorkloadCheck bool `json:"skipExistingWorkloadCheck,omitempty"`
	IncludeEtcd               bool `json:"includeEtcd,omitempty"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Install   Install   `json:"install,omitempty"`
	Uninstall Uninstall `json:"uninstall,omitempty"`
}

// GetOperatorNamespace tries to figure out operator namespace
func (spec *KubectlStorageOSConfigSpec) GetOperatorNamespace() (namespace string) {
	namespace = spec.Install.StorageOSOperatorNamespace
	if namespace == "" {
		namespace = spec.Uninstall.StorageOSOperatorNamespace
	}

	return
}

// KubectlStorageOSConfigStatus defines the observed state of KubectlStorageOSConfig
type KubectlStorageOSConfigStatus struct {
}

// Install defines options for cli install subcommand
type Install struct {
	Wait                        bool   `json:"wait,omitempty"`
	StorageOSVersion            string `json:"storageOSVersion,omitempty"`
	StorageOSOperatorNamespace  string `json:"storageOSOperatorNamespace,omitempty"`
	StorageOSClusterNamespace   string `json:"storageOSClusterNamespace,omitempty"`
	EtcdNamespace               string `json:"etcdNamespace,omitempty"`
	StorageOSOperatorYaml       string `json:"storageOSOperatorYaml,omitempty"`
	StorageOSClusterYaml        string `json:"storageOSClusterYaml,omitempty"`
	ResourceQuotaYaml           string `json:"resourceQuotaYaml,omitempty"`
	EtcdOperatorYaml            string `json:"etcdOperatorYaml,omitempty"`
	EtcdClusterYaml             string `json:"etcdClusterYaml,omitempty"`
	EtcdEndpoints               string `json:"etcdEndpoints,omitempty"`
	EtcdTLSEnabled              bool   `json:"etcdTLSEnabled,omitempty"`
	EtcdSecretName              string `json:"etcdSecretName,omitempty"`
	SkipEtcdEndpointsValidation bool   `json:"skipEtcdEndpointsValidation,omitempty"`
	EtcdStorageClassName        string `json:"etcdStorageClassName,omitempty"`
	AdminUsername               string `json:"adminUsername,omitempty"`
	AdminPassword               string `json:"adminPassword,omitempty"`
	PortalKeyPath               string `json:"portalKeyPath,omitempty"`
}

// Uninstall defines options for cli uninstall subcommand
type Uninstall struct {
	StorageOSOperatorNamespace string `json:"storageOSOperatorNamespace,omitempty"`
	EtcdNamespace              string `json:"etcdNamespace,omitempty"`
}

type InstallerMeta struct {
	StorageOSSecretYaml string `json:"storageOSSecretYaml,omitempty"`
	SecretName          string `json:"secretName,omitempty"`
	SecretNamespace     string `json:"secretNamespace,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KubectlStorageOSConfig is the Schema for the kubectlstorageosconfigs API
type KubectlStorageOSConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec          KubectlStorageOSConfigSpec   `json:"spec,omitempty"`
	Status        KubectlStorageOSConfigStatus `json:"status,omitempty"`
	InstallerMeta InstallerMeta                `json:"installerMeta,omitempty"`
}

//+kubebuilder:object:root=true

// KubectlStorageOSConfigList contains a list of KubectlStorageOSConfig
type KubectlStorageOSConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubectlStorageOSConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubectlStorageOSConfig{}, &KubectlStorageOSConfigList{})
}
