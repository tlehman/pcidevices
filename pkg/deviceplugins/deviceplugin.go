package deviceplugins

import (
	"fmt"

	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
)

// Add a PCI Device to the PCIDevicePlugin for that resourceName. When the calling function uses the new PCIDevicePlugin,
// all pointers to the old one will be set to the new one, and the gc will destroy the old one.
func (oldDp *PCIDevicePlugin) AddPCIDevicePlugin(resourceName string, claim *v1beta1.PCIDeviceClaim) *PCIDevicePlugin {
	pciDevice := &PCIDevice{
		pciID:      claim.Spec.Address,
		driver:     claim.Status.KernelDriverToUnbind,
		pciAddress: claim.Spec.Address,
	}

	pciDevices := append(oldDp.pcidevs, pciDevice)
	newDp := NewPCIDevicePlugin(pciDevices, resourceName)
	return newDp
}

// Looks for a PCIDevicePlugin with that resourceName, and returns it, or an error if it doesn't exist
func Find(
	resourceName string,
	dps map[string]*PCIDevicePlugin,
) (*PCIDevicePlugin, error) {
	dp, found := dps[resourceName]
	if !found {
		return nil, fmt.Errorf("no device plugin found for resource %s", resourceName)
	}
	return dp, nil
}

// Creates a new PCIDevicePlugin with that resourceName, and returns it
func Create(
	resourceName string,
	claim *v1beta1.PCIDeviceClaim,
) *PCIDevicePlugin {
	// Check if there are any PCIDevicePlugins with that resourceName
	pcidevs := []*PCIDevice{{
		pciID:      claim.Spec.Address,
		driver:     claim.Status.KernelDriverToUnbind,
		pciAddress: claim.Spec.Address,
	}}
	// Create the DevicePlugin
	dp := NewPCIDevicePlugin(pcidevs, resourceName)
	return dp
}

// Removes a PCIDevice from the PCIDevicePlugin for that resourceName
func Remove(
	resourceName string,
	claim *v1beta1.PCIDeviceClaim,
	dps map[string]*PCIDevicePlugin,
) (*PCIDevicePlugin, error) {
	dp, err := Find(resourceName, dps)
	if err != nil {
		return nil, err
	}
	// pull out the pcidevs, delete and return a new one
	pcidevs := dp.GetPCIDevices()
	// find the index of the pcidev to remove
	for i, pcidev := range pcidevs {
		if pcidev.GetID() == claim.Spec.Address {
			pcidevs = append(pcidevs[:i], pcidevs[i+1:]...)
			break
		}
	}
	dp = NewPCIDevicePlugin(pcidevs, resourceName)

	return dp, nil
}
