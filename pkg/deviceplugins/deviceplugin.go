package deviceplugins

import (
	"fmt"

	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	"github.com/sirupsen/logrus"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

// Add a PCI Device to the PCIDevicePlugin for that resourceName
func (dp *PCIDevicePlugin) MarkPCIDeviceAsHealthy(resourceName string, claim *v1beta1.PCIDeviceClaim) {
	logrus.Infof(
		"[AddPCIDeviceToPlugin] Marking pcidevice: %s in device plugin: %s as healthy",
		claim.Spec.Address,
		resourceName,
	)

	dp.lock.Lock()
	defer dp.lock.Unlock()
	for i := 0; i < len(dp.devs); i++ {
		logrus.Infof("[AddPCIDeviceToPlugin] dp.devs[%d].ID = %s", i, dp.devs[i].ID)
		if dp.devs[i].ID == claim.Spec.Address {
			dp.devs[i] = &pluginapi.Device{
				ID:     claim.Spec.Address,
				Health: pluginapi.Healthy,
			}
		}
		logrus.Infof("[AddPCIDeviceToPlugin] dp.devs[%d].Health = %s", i, dp.devs[i].Health)
	}
}

// Remove a PCI Device from the PCIDevicePlugin
func (dp *PCIDevicePlugin) RemovePCIDeviceFromPlugin(claim *v1beta1.PCIDeviceClaim) error {
	for i, pcidev := range dp.pcidevs {
		if pcidev.pciID == claim.Spec.Address {
			dp.lock.Lock()
			defer dp.lock.Unlock()
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
	pdsWithSameResourceName []*v1beta1.PCIDevice,
) *PCIDevicePlugin {
	// Check if there are any PCIDevicePlugins with that resourceName
	pcidevs := []*PCIDevice{}
	for _, pd := range pdsWithSameResourceName {
		pcidevs = append(pcidevs, &PCIDevice{
			pciID:      pd.Status.Address,
			driver:     pd.Status.KernelDriverInUse,
			pciAddress: pd.Status.Address, // this redundancy is here to distinguish between the ID and the PCI Address. They have the same value but mean different things
		})
	}
	// Create the DevicePlugin
	dp := NewPCIDevicePlugin(pcidevs, resourceName)
	// Mark only the claimed device as healthy
	for _, dev := range dp.devs {
		if dev.ID == claim.Spec.Address {
			dev.Health = pluginapi.Healthy // 'healthy' devices are ones that are enabled for passthrough
		} else {
			dev.Health = pluginapi.Unhealthy
		}
	}
	return dp
}
