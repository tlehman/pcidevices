package deviceplugins

import (
	"github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	"github.com/sirupsen/logrus"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

func (dp *PCIDevicePlugin) MarkPCIDeviceAsHealthy(resourceName string, claim *v1beta1.PCIDeviceClaim) {
	go func() {
		dp.health <- deviceHealth{
			DevId:  claim.Spec.Address,
			Health: pluginapi.Healthy,
		}
	}()
}

func (dp *PCIDevicePlugin) MarkPCIDeviceAsUnhealthy(claim *v1beta1.PCIDeviceClaim) {
	go func() {
		dp.health <- deviceHealth{
			DevId:  claim.Spec.Address,
			Health: pluginapi.Unhealthy,
		}
	}()
}

// Looks for a PCIDevicePlugin with that resourceName, and returns it, or an error if it doesn't exist
func Find(
	resourceName string,
	dps map[string]*PCIDevicePlugin,
) *PCIDevicePlugin {
	dp, found := dps[resourceName]
	if !found {
		return nil
	}
	return dp
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

// This function adds the PCIDevice to the device plugin, or creates the device plugin if it doesn't exist
func (dp *PCIDevicePlugin) AddDevice(pd *v1beta1.PCIDevice, pdc *v1beta1.PCIDeviceClaim) error {
	var err error
	resourceName := pd.Status.ResourceName
	if dp != nil {
		logrus.Infof("Adding new claimed %s to device plugin", resourceName)
		dp.MarkPCIDeviceAsHealthy(resourceName, pdc)
	}
	return err
}

// This function adds the PCIDevice to the device plugin, or creates the device plugin if it doesn't exist
func (dp *PCIDevicePlugin) RemoveDevice(pd *v1beta1.PCIDevice, pdc *v1beta1.PCIDeviceClaim) error {
	var err error
	resourceName := pd.Status.ResourceName
	if dp != nil {
		logrus.Infof("Removing %s from device plugin", resourceName)
		dp.MarkPCIDeviceAsUnhealthy(pdc)
	}
	return err
}
