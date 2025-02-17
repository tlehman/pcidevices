package pcidevice

import (
	"context"
	"fmt"
	"os"
	"time"

	ctlnetwork "github.com/harvester/harvester-network-controller/pkg/generated/controllers/network.harvesterhci.io"
	ctlnetworkv1beta1 "github.com/harvester/harvester-network-controller/pkg/generated/controllers/network.harvesterhci.io/v1beta1"
	"github.com/jaypipes/ghw"
	ctlcore "github.com/rancher/wrangler/pkg/generated/controllers/core"
	ctlcorev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	v1beta1 "github.com/harvester/pcidevices/pkg/apis/devices.harvesterhci.io/v1beta1"
	ctl "github.com/harvester/pcidevices/pkg/generated/controllers/devices.harvesterhci.io/v1beta1"
	"github.com/harvester/pcidevices/pkg/iommu"
	"github.com/harvester/pcidevices/pkg/util/nichelper"
)

const (
	reconcilePeriod  = time.Second * 20
	pciBridgeClassID = "0604"
)

type Handler struct {
	client          ctl.PCIDeviceClient
	pci             *ghw.PCIInfo
	nodeCache       ctlcorev1.NodeCache
	vlanConfigCache ctlnetworkv1beta1.VlanConfigCache
	skipAddresses   []string
}

func Register(
	ctx context.Context,
	pd ctl.PCIDeviceClient,
	coreFactory *ctlcore.Factory,
	networkFactory *ctlnetwork.Factory) error {
	logrus.Info("Registering PCI Devices controller")

	handler := &Handler{
		client:          pd,
		nodeCache:       coreFactory.Core().V1().Node().Cache(),
		vlanConfigCache: networkFactory.Network().V1beta1().VlanConfig().Cache(),
	}

	nodename := os.Getenv("NODE_NAME")

	if err := coreFactory.Sync(ctx); err != nil {
		return fmt.Errorf("error waiting for coreFactory to sync")
	}

	if err := networkFactory.Sync(ctx); err != nil {
		return fmt.Errorf("error waiting for coreFactory to sync")
	}

	// start goroutine to regularly reconcile the PCI Devices list
	ticker := time.NewTicker(reconcilePeriod)
	for range ticker.C {
		logrus.Info("Reconciling PCI Devices list")
		pci, err := ghw.PCI()
		if err != nil {
			return fmt.Errorf("error listing pcidevices: %v", err)
		}
		skipAddresses, err := nichelper.IdentifyHarvesterManagedNIC(nodename, handler.nodeCache, handler.vlanConfigCache)
		if err != nil {
			return fmt.Errorf("error querying management nic pci addresses: %v", err)
		}
		pciBridgeAddresses := identifyPCIBridgeDevices(pci)
		skipAddresses = append(skipAddresses, pciBridgeAddresses...)
		handler.pci = pci
		handler.skipAddresses = skipAddresses
		if err := handler.reconcilePCIDevices(nodename); err != nil {
			logrus.Errorf("PCI device reconciliation error: %v", err)
			return err
		}
	}
	return nil
}

func (h *Handler) reconcilePCIDevices(nodename string) error {
	// Build up the IOMMU group map
	iommuGroupPaths, err := iommu.GroupPaths()
	if err != nil {
		return err
	}
	iommuGroupMap := iommu.GroupMapForPCIDevices(iommuGroupPaths)

	commonLabels := map[string]string{"nodename": nodename} // label
	var setOfRealPCIAddrs map[string]bool = make(map[string]bool)
	for _, dev := range h.pci.Devices {
		if !containsString(h.skipAddresses, dev.Address) {
			setOfRealPCIAddrs[dev.Address] = true
			name := v1beta1.PCIDeviceNameForHostname(dev, nodename)
			// Check if device is stored
			devCR, err := h.client.Get(name, metav1.GetOptions{})

			if err != nil {
				if apierrors.IsNotFound(err) {
					logrus.Infof("[PCIDeviceController] Device %s does not exist", name)

					// Create the PCIDevice CR if it doesn't exist
					var pdToCreate v1beta1.PCIDevice = v1beta1.NewPCIDeviceForHostname(dev, nodename)
					logrus.Infof("Creating PCI Device: %s\n", err)
					pdToCreate.Labels = commonLabels
					devCR, err = h.client.Create(&pdToCreate)
					if err != nil {
						logrus.Errorf("[PCIDeviceController] Failed to create PCI Device: %v", err)
						return err
					}
				} else {
					logrus.Errorf("[PCIDeviceController] error fetching device %s: %v", name, err)
					return err
				}

			}

			devCopy := devCR.DeepCopy()

			// during reboot if the device driver has changed back from vfio, then update the CRD
			// to correct driver in use. This will ensure that the original driver is correctly updated on device
			// the PCIDeviceClaim checks for driver to identify if a rebind is needed on reboot
			if devCopy.Status.KernelDriverInUse != dev.Driver {
				devCopy.Status.KernelDriverInUse = dev.Driver
			}
			// Update only modifies the status, no need to update the main object
			devCopy.Status.Update(dev, nodename, iommuGroupMap) // update the in-memory CR with the current PCI info
			_, err = h.client.UpdateStatus(devCopy)
			if err != nil {
				logrus.Errorf("[PCIDeviceController] Failed to update status sub-resource: %v", err)
				return err
			}
		}

	}

	// remove non-existent devices
	selector := labels.SelectorFromValidatedSet(commonLabels)

	pdList, err := h.client.List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		logrus.Errorf("[PCIDeviceController] error listing devices for node %s: %v", nodename, err)
		return err
	}

	var deleteList []v1beta1.PCIDevice

	for _, v := range pdList.Items {
		if ok := setOfRealPCIAddrs[v.Status.Address]; !ok {
			deleteList = append(deleteList, v)
		}
	}

	for _, v := range deleteList {
		if err := h.client.Delete(v.Name, &metav1.DeleteOptions{}); err != nil {
			logrus.Errorf("[PCIDeviceController] Faield to delete non existent device: %s on node %s", v.Name, v.Status.NodeName)
			return err
		}
	}

	return nil
}

func containsString(elements []string, element string) bool {
	for _, v := range elements {
		if v == element {
			return true
		}
	}

	return false
}

// identifyPCIBridgeDevices will identify devices which are pci bridges to skip the same
// as these cannot be bound to vfio-pci though share the same iommu group with devices attached
// to the brdige
func identifyPCIBridgeDevices(pci *ghw.PCIInfo) []string {
	var pciBridgeAddresses []string
	for _, v := range pci.Devices {
		if fmt.Sprintf("%s%s", v.Class.ID, v.Subclass.ID) == pciBridgeClassID {
			pciBridgeAddresses = append(pciBridgeAddresses, v.Address)
		}
	}
	return pciBridgeAddresses
}
