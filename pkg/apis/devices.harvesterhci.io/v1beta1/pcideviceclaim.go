package v1beta1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// a PCIDeviceClaim is used to reserve a PCI Device for a single
type PCIDeviceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PCIDeviceClaimSpec   `json:"spec,omitempty"`
	Status PCIDeviceClaimStatus `json:"status,omitempty"`
}

type PCIDeviceClaimSpec struct {
	Address  string `json:"address"`
	NodeName string `json:"nodeName"`
	UserName string `json:"userName"`
}

func (s PCIDeviceClaimSpec) NodeAddr() string {
	return fmt.Sprintf("%s-%s", s.NodeName, s.Address)
}

type PCIDeviceClaimStatus struct {
	KernelDriverToUnbind string `json:"kernelDriverToUnbind"`
	PassthroughEnabled   bool   `json:"passthroughEnabled"`

	// StateBeforePassthroughEnabled was created to solve issue #3651, which
	// concerned NIC devices. When a NIC device has a link state = UP and then
	// is enabled for passthrough, then disabled for passthrough, it has a link
	// state = DOWN. In order to restore the PCIDevice to the original state,
	// We need a place to store that state prior to the device being enabled.
	// For NICs, this will be the ethernet link state
	// https://github.com/harvester/harvester/issues/3651
	StateBeforePassthroughEnabled string `json:"stateBeforePassthroughEnabled"`
}
