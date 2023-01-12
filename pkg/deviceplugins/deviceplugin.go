package deviceplugins

import (
	"fmt"

	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	"github.com/sirupsen/logrus"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

func (dp *PCIDevicePlugin) MarkPCIDeviceAsHealthy(resourceName string, claim *v1beta1.PCIDeviceClaim) {
	logrus.Infof(
		"[MarkPCIDeviceAsHealthy] Marking pcidevice: %s in device plugin: %s as healthy",
		claim.Spec.Address,
		resourceName,
	)

	dp.lock.Lock()
	defer dp.lock.Unlock()
	for i := 0; i < len(dp.devs); i++ {
		logrus.Infof("[MarkPCIDeviceAsHealthy] dp.devs[%d].ID = %s", i, dp.devs[i].ID)
		if dp.devs[i].ID == claim.Spec.Address {
			dp.devs[i] = &pluginapi.Device{
				ID:     claim.Spec.Address,
				Health: pluginapi.Healthy,
			}
		}
		logrus.Infof("[MarkPCIDeviceAsHealthy] dp.devs[%d].Health = %s", i, dp.devs[i].Health)
	}
	// For after initialization
	if dp.initialized {
		go func() {
			dp.health <- deviceHealth{
				DevId:  claim.Spec.Address,
				Health: pluginapi.Healthy,
			}
		}()
	}
}

func (dp *PCIDevicePlugin) MarkPCIDeviceAsUnhealthy(claim *v1beta1.PCIDeviceClaim) error {
	for i, dev := range dp.devs {
		if dev.ID == claim.Spec.Address {
			dp.lock.Lock()
			defer dp.lock.Unlock()
			dp.devs[i].Health = pluginapi.Healthy
			return nil
		}
	}
	return fmt.Errorf("[MarkPCIDeviceAsUnhealthy] device plugin does not have PCI device %s", claim.Spec.Address)
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
	dp.MarkPCIDeviceAsHealthy(resourceName, claim)
	return dp
}
