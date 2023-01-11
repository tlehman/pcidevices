package deviceplugins

import (
	"fmt"

	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	"github.com/sirupsen/logrus"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

// Add a PCI Device to the PCIDevicePlugin for that resourceName
func (dp *PCIDevicePlugin) AddPCIDeviceToPlugin(resourceName string, claim *v1beta1.PCIDeviceClaim) {
	pciDevice := &PCIDevice{
		pciID:      claim.Spec.Address,
		driver:     claim.Status.KernelDriverToUnbind,
		pciAddress: claim.Spec.Address,
	}
	logrus.Infof("[AddPCIDeviceToPlugin] Adding pcidevice: %s to device plugin: %s", pciDevice.pciAddress, resourceName)
	logrus.Infof("[AddPCIDeviceToPlugin] before, len(dp.devs): %d", len(dp.devs))

	dp.pcidevs = append(dp.pcidevs, pciDevice)
	// Reconstruct the devs from the pcidevs
	dp.devs = []*pluginapi.Device{}
	dp.devs = constructDPIdevices(dp.pcidevs, dp.iommuToPCIMap)
	logrus.Infof("[AddPCIDeviceToPlugin] after, len(dp.devs): %d", len(dp.devs))
}

// Remove a PCI Device from the PCIDevicePlugin
func (dp *PCIDevicePlugin) RemovePCIDeviceFromPlugin(claim *v1beta1.PCIDeviceClaim) error {
	for i, pcidev := range dp.pcidevs {
		if pcidev.pciID == claim.Spec.Address {
			dp.pcidevs = append(dp.pcidevs[:i], dp.pcidevs[i+1:]...)
			// Reconstruct the devs from the pcidevs
			dp.devs = []*pluginapi.Device{}
			dp.devs = constructDPIdevices(dp.pcidevs, dp.iommuToPCIMap)
			return nil
		}
	}
	return fmt.Errorf("[RemovePCIDeviceFromPlugin] device plugin does not have PCI device %s", claim.Spec.Address)
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
