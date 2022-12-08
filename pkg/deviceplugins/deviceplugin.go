package deviceplugins

import (
	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	dm "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

/* Wrapper for dm.PCIDevicePlugin to retain access to the devs array
 */
type PCIDevicePlugin struct {
	dp         *dm.PCIDevicePlugin
	pciDevices []*dm.PCIDevice
}

// Add a PCI Device to the PCIDevicePlugin for that resourceName. When the calling function uses the new PCIDevicePlugin,
// all pointers to the old one will be set to the new one, and the gc will destroy the old one.
func (oldDp *PCIDevicePlugin) AddPCIDevicePlugin(resourceName string, claim *v1beta1.PCIDeviceClaim) *PCIDevicePlugin {
	pciDevice := &dm.PCIDevice{
		pciID:      claim.Spec.Address,
		driver:     claim.Status.KernelDriverToUnbind,
		pciAddress: claim.Spec.Address,
	}
	var pciDevices []*dm.PCIDevice = append(oldDp.pciDevices, pciDevice)
	newDp := dm.NewPCIDevicePlugin(pciDevices, resourceName)

	return &PCIDevicePlugin{
		dp:         newDp,
		pciDevices: pciDevices,
	}
}

func NewPCIDevicePlugin(resourceName string, claim *v1beta1.PCIDeviceClaim) *PCIDevicePlugin {
	pciDevice := &dm.PCIDevice{
		pciID:      claim.Spec.Address,
		driver:     claim.Status.KernelDriverToUnbind,
		pciAddress: claim.Spec.Address,
	}
	var pciDevices []*dm.PCIDevice = []*dm.PCIDevice{pciDevice}
	newDp := dm.NewPCIDevicePlugin(pciDevices, resourceName)

	return &PCIDevicePlugin{
		dp:         newDp,
		pciDevices: pciDevices,
	}
}

/* This function takes a PCIDeviceClaim, then checks if there are any PCIDevicePlugins with that resourceName,
* if there are, it destroys it and creates a new one,
* otherwise it just creates a new one.

claim is the new PCIDeviceClaim,
dps is the map from resourceName => all the PCIDevicePlugins
*/
func FindOrCreateDevicePluginFromPCIDeviceClaim(
	resourceName string,
	claim *v1beta1.PCIDeviceClaim,
	dps map[string]*PCIDevicePlugin,
) *PCIDevicePlugin {
	// Check if there are any PCIDevicePlugins with that resourceName
	dp, found := dps[resourceName]
	if !found {
		// Create the DevicePlugin
		dp = NewPCIDevicePlugin(resourceName, claim)
		dps[resourceName] = dp
	} else {
		// Destroy the old DevicePlugin and create a new one
		dps[resourceName] = dp.AddPCIDevicePlugin(resourceName, claim)
	}
	return dp
}
