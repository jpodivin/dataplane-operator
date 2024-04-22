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

package v1beta1

import (
	"fmt"
	"regexp"

	"golang.org/x/exp/slices"

	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	baremetalv1 "github.com/openstack-k8s-operators/openstack-baremetal-operator/api/v1beta1"
	openstackv1 "github.com/openstack-k8s-operators/openstack-operator/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// OpenStackDataPlaneNodeSetSpec defines the desired state of OpenStackDataPlaneNodeSet
type OpenStackDataPlaneNodeSetSpec struct {
	// +kubebuilder:validation:Optional
	// BaremetalSetTemplate Template for BaremetalSet for the NodeSet
	BaremetalSetTemplate baremetalv1.OpenStackBaremetalSetSpec `json:"baremetalSetTemplate,omitempty"`

	// +kubebuilder:validation:Required
	// NodeTemplate - node attributes specific to nodes defined by this resource. These
	// attributes can be overriden at the individual node level, else take their defaults
	// from valus in this section.
	NodeTemplate NodeTemplate `json:"nodeTemplate"`

	// Nodes - Map of Node Names and node specific data. Values here override defaults in the
	// upper level section.
	// +kubebuilder:validation:Required
	Nodes map[string]NodeSection `json:"nodes"`

	// SecretMaxSize - Maximum size in bytes of a Kubernetes secret. This size is currently situated around
	// 1 MiB (nearly 1 MB).
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1048576
	SecretMaxSize int `json:"secretMaxSize" yaml:"secretMaxSize"`

	// +kubebuilder:validation:Optional
	//
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// PreProvisioned - Set to true if the nodes have been Pre Provisioned.
	PreProvisioned bool `json:"preProvisioned,omitempty"`

	// Env is a list containing the environment variables to pass to the pod
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to pass to the ansibleee resource
	// which allows to connect the ansibleee runner to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={download-cache,bootstrap,configure-network,validate-network,install-os,configure-os,ssh-known-hosts,run-os,reboot-os,install-certs,ovn,neutron-metadata,libvirt,nova,telemetry}
	// Services list
	Services []string `json:"services"`

	// TLSEnabled - Whether the node set has TLS enabled.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	TLSEnabled bool `json:"tlsEnabled" yaml:"tlsEnabled"`

	// Tags - Additional tags for NodeSet
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Data Plane NodeSet"
//+kubebuilder:resource:shortName=osdpns;osdpnodeset;osdpnodesets
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackDataPlaneNodeSet is the Schema for the openstackdataplanenodesets API
// OpenStackDataPlaneNodeSet name must be a valid RFC1123 as it is used in labels
type OpenStackDataPlaneNodeSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackDataPlaneNodeSetSpec   `json:"spec,omitempty"`
	Status OpenStackDataPlaneNodeSetStatus `json:"status,omitempty"`
}

// OpenStackDataPlaneNodeSetStatus defines the observed state of OpenStackDataPlaneNodeSet
type OpenStackDataPlaneNodeSetStatus struct {
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// Deployed
	Deployed bool `json:"deployed,omitempty" optional:"true"`

	// DeploymentStatuses
	DeploymentStatuses map[string]condition.Conditions `json:"deploymentStatuses,omitempty" optional:"true"`

	// DNSClusterAddresses
	DNSClusterAddresses []string `json:"dnsClusterAddresses,omitempty" optional:"true"`

	// CtlplaneSearchDomain
	CtlplaneSearchDomain string `json:"ctlplaneSearchDomain,omitempty" optional:"true"`

	// AllHostnames
	AllHostnames map[string]map[infranetworkv1.NetNameStr]string `json:"allHostnames,omitempty" optional:"true"`

	// AllIPs
	AllIPs map[string]map[infranetworkv1.NetNameStr]string `json:"allIPs,omitempty" optional:"true"`

	// ConfigMapHashes
	ConfigMapHashes map[string]string `json:"configMapHashes,omitempty" optional:"true"`

	// SecretHashes
	SecretHashes map[string]string `json:"secretHashes,omitempty" optional:"true"`

	// ConfigHash - holds the curret hash of the NodeTemplate and Node sections of the struct.
	// This hash is used to determine when new Ansible executions are required to roll
	// out config changes.
	ConfigHash string `json:"configHash,omitempty"`

	// DeployedConfigHash - holds the hash of the NodeTemplate and Node sections of the struct
	// that was last deployed.
	// This hash is used to determine when new Ansible executions are required to roll
	// out config changes.
	DeployedConfigHash string `json:"deployedConfigHash,omitempty"`

	//ObservedGeneration - the most recent generation observed for this NodeSet. If the observed generation is less than the spec generation, then the controller has not processed the latest changes.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// DeployedVersion
	DeployedVersion string `json:"deployedVersion,omitempty"`

	// ContainerImages
	ContainerImages map[string]string `json:"containerImages,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true

// OpenStackDataPlaneNodeSetList contains a list of OpenStackDataPlaneNodeSets
type OpenStackDataPlaneNodeSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackDataPlaneNodeSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenStackDataPlaneNodeSet{}, &OpenStackDataPlaneNodeSetList{})
}

// IsReady - returns true if the DataPlane is ready
func (instance OpenStackDataPlaneNodeSet) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

// InitConditions - Initializes Status Conditons
func (instance *OpenStackDataPlaneNodeSet) InitConditions() {
	instance.Status.Conditions = condition.Conditions{}
	instance.Status.DeploymentStatuses = make(map[string]condition.Conditions)

	cl := condition.CreateList(
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(SetupReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(NodeSetIPReservationReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(NodeSetDNSDataReadyCondition, condition.InitReason, condition.InitReason),
		condition.UnknownCondition(condition.ServiceAccountReadyCondition, condition.InitReason, condition.InitReason),
	)

	// Only set Baremetal related conditions if we have baremetal hosts included in the
	// baremetalSetTemplate.
	if len(instance.Spec.BaremetalSetTemplate.BaremetalHosts) > 0 {
		cl = append(cl, *condition.UnknownCondition(NodeSetBareMetalProvisionReadyCondition, condition.InitReason, condition.InitReason))
	}

	instance.Status.Conditions.Init(&cl)
	instance.Status.Deployed = false
}

// GetAnsibleEESpec - get the fields that will be passed to AEE
func (instance OpenStackDataPlaneNodeSet) GetAnsibleEESpec() AnsibleEESpec {
	return AnsibleEESpec{
		NetworkAttachments: instance.Spec.NetworkAttachments,
		ExtraMounts:        instance.Spec.NodeTemplate.ExtraMounts,
		Env:                instance.Spec.Env,
		ServiceAccountName: instance.Name,
	}
}

// ContainerImageDefaults - the hardcoded defaults which are the last fallback
// if no values are set elsewhere.
var ContainerImageDefaults = openstackv1.ContainerImages{
	ContainerTemplate: openstackv1.ContainerTemplate{
		EdpmFrrImage:                  getStrPtr("quay.io/podified-antelope-centos9/openstack-frr:current-podified"),
		EdpmIscsidImage:               getStrPtr("quay.io/podified-antelope-centos9/openstack-iscsid:current-podified"),
		EdpmLogrotateCrondImage:       getStrPtr("quay.io/podified-antelope-centos9/openstack-cron:current-podified"),
		EdpmNeutronMetadataAgentImage: getStrPtr("quay.io/podified-antelope-centos9/openstack-neutron-metadata-agent-ovn:current-podified"),
		EdpmNeutronSriovAgentImage:    getStrPtr("quay.io/podified-antelope-centos9/openstack-neutron-sriov-agent:current-podified"),
		EdpmMultipathdImage:           getStrPtr("quay.io/podified-antelope-centos9/openstack-multipathd:current-podified"),
		NovaComputeImage:              getStrPtr("quay.io/podified-antelope-centos9/openstack-nova-compute:current-podified"),
		OvnControllerImage:            getStrPtr("quay.io/podified-antelope-centos9/openstack-ovn-controller:current-podified"),
		EdpmOvnBgpAgentImage:          getStrPtr("quay.io/podified-antelope-centos9/openstack-ovn-bgp-agent:current-podified"),
		CeilometerComputeImage:        getStrPtr("quay.io/podified-antelope-centos9/openstack-telemetry-ceilometer-compute:current-podified"),
		CeilometerIpmiImage:           getStrPtr("quay.io/podified-antelope-centos9/openstack-telemetry-ceilometer-ipmi:current-podified"),
		EdpmNodeExporterImage:         getStrPtr("quay.io/podified-antelope-centos9/openstack-telemetry-node-exporter:current-podified"),
	}}

// ContainerImages - the values if no OpenStackVersion is used
var ContainerImages openstackv1.ContainerImages

// SetupDefaults - initializes any CRD field defaults based on environment variables
// called from main.go
func SetupDefaults() {
	// Acquire environmental defaults and initialize dataplane defaults with them
	ContainerImages = openstackv1.ContainerImages{
		ContainerTemplate: openstackv1.ContainerTemplate{
			EdpmFrrImage:                  getImageDefault("RELATED_IMAGE_EDPM_FRR_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmFrrImage),
			EdpmIscsidImage:               getImageDefault("RELATED_IMAGE_EDPM_ISCSID_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmIscsidImage),
			EdpmLogrotateCrondImage:       getImageDefault("RELATED_IMAGE_EDPM_LOGROTATE_CROND_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmLogrotateCrondImage),
			EdpmMultipathdImage:           getImageDefault("RELATED_IMAGE_EDPM_MULTIPATHD_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmMultipathdImage),
			EdpmNeutronMetadataAgentImage: getImageDefault("RELATED_IMAGE_EDPM_NEUTRON_METADATA_AGENT_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmNeutronMetadataAgentImage),
			EdpmNeutronSriovAgentImage:    getImageDefault("RELATED_IMAGE_EDPM_NEUTRON_SRIOV_AGENT_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmNeutronSriovAgentImage),
			EdpmNodeExporterImage:         getImageDefault("RELATED_IMAGE_EDPM_NODE_EXPORTER_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmNodeExporterImage),
			EdpmOvnBgpAgentImage:          getImageDefault("RELATED_IMAGE_EDPM_OVN_BGP_AGENT_IMAGE_URL_DEFAULT", ContainerImageDefaults.EdpmOvnBgpAgentImage),
			CeilometerComputeImage:        getImageDefault("RELATED_IMAGE_CEILOMETER_COMPUTE_IMAGE_URL_DEFAULT", ContainerImageDefaults.CeilometerComputeImage),
			CeilometerIpmiImage:           getImageDefault("RELATED_IMAGE_CEILOMETER_IPMI_IMAGE_URL_DEFAULT", ContainerImageDefaults.CeilometerIpmiImage),
			NovaComputeImage:              getImageDefault("RELATED_IMAGE_NOVA_COMPUTE_IMAGE_URL_DEFAULT", ContainerImageDefaults.NovaComputeImage),
			OvnControllerImage:            getImageDefault("RELATED_IMAGE_OVN_CONTROLLER_AGENT_IMAGE_URL_DEFAULT", ContainerImageDefaults.OvnControllerImage),
		},
	}
}

func getImageDefault(envVar string, defaultImage *string) *string {
	d := util.GetEnvVar(envVar, *defaultImage)
	return &d
}

func getStrPtr(in string) *string {
	return &in
}

// duplicateNodeCheck checks the NodeSetList for pre-existing nodes. If the user is trying to redefine an
// existing node, we will return an error and block resource creation.
func (r *OpenStackDataPlaneNodeSetSpec) duplicateNodeCheck(nodeSetList *OpenStackDataPlaneNodeSetList) (errors field.ErrorList) {
	existingNodeNames := make([]string, 0)
	for _, existingNode := range nodeSetList.Items {
		for _, node := range existingNode.Spec.Nodes {
			existingNodeNames = append(existingNodeNames, node.HostName)
			if node.Ansible.AnsibleHost != "" {
				existingNodeNames = append(existingNodeNames, node.Ansible.AnsibleHost)
			}
		}
	}

	for _, newNodeName := range r.Nodes {
		if slices.Contains(existingNodeNames, newNodeName.HostName) || slices.Contains(existingNodeNames, newNodeName.Ansible.AnsibleHost) {
			errors = append(errors, field.Invalid(
				field.NewPath("spec").Child("nodes"),
				newNodeName,
				fmt.Sprintf("node %s already exists in another cluster", newNodeName.HostName)))
		}
	}

	return
}

// NodeHostNameIsFQDN Helper to check if a hostname is fqdn
func NodeHostNameIsFQDN(hostname string) bool {
	// Regular expression to match a valid FQDN
	// This regex assumes that the hostname and domain name segments only contain letters, digits, hyphens, and periods.
	regex := `^([a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+\.[a-zA-Z]{2,}$`

	match, _ := regexp.MatchString(regex, hostname)
	return match
}
