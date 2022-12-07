package deviceplugins

import (
	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	dm "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"

)

/* This function takes a PCIDeviceClaim, then checks if there are any PCIDevicePlugins with that resourceName,
* if there are, it destroys it and creates a new one,
* otherwise it just creates a new one.

claim is the new PCIDeviceClaim,
dps is the map from resourceName => all the PCIDevicePlugins
*/
func NewPCIDevicePluginFromClaim(claim *v1beta1.PCIDeviceClaim, dps map[string]*dm.PCIDevicePlugin) (*dm.PCIDevicePlugin, error) {
	// Check if there are any PCIDevicePlugins with that resourceName
	pd := claim.OwnerReferences[0].Name
	// TODO Need the resourceName on the PCIDeviceClaim

	for _, dp := range dps {
		if dp.ResourceName == resourceName {
			devs := dp.GetDevices()
			dev := pluginapi.Device{
				ID: claim.Spec.Address,
				Health: "Healthy",
				Topology: ,
			}
			devs = append(devs, pluginapi.
			// Destroy the PCIDevicePlugin and create a new one
			dp.Destroy()
			return dm.NewPCIDevicePlugin(devs, resourceName)
		}
	}

}
