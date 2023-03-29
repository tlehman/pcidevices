package nichelper

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	ctlnetworkv1beta1 "github.com/harvester/harvester-network-controller/pkg/generated/controllers/network.harvesterhci.io/v1beta1"
	"github.com/jaypipes/ghw"
	ctlcorev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/json"
)

const (
	defaultBRInterface     = "mgmt-br"
	defaultBOInterface     = "mgmt-bo"
	matchedNodesAnnotation = "network.harvesterhci.io/matched-nodes"
)

func IdentifyHarvesterManagedNIC(nodeName string, nodeCache ctlcorev1.NodeCache, vlanConfigCache ctlnetworkv1beta1.VlanConfigCache) ([]string, error) {
	link, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("error fetching link by name: %v", err)
	}

	// masterBondedIndexes will contain index id for the default bonded interfaces in harvester
	var masterBondedIndexes []int

	logrus.Debug("listing link information")
	for _, l := range link {
		if l.Attrs().Name == defaultBRInterface || l.Attrs().Name == defaultBOInterface {
			masterBondedIndexes = append(masterBondedIndexes, l.Attrs().Index)
		}
	}

	// skipInterfaces contains names of interfaces using in the default harvester bonding interfaces
	// these are to be used by the PCIDevices controller to skip said devices
	var skipInterfaces []string
	for _, i := range masterBondedIndexes {
		for _, l := range link {
			// check helps skip over cases when mgmt-br is also being used for vm networks
			// in which case mgmt-bo is also pointing to mgmt-br
			if l.Attrs().Slave != nil {
				if l.Attrs().MasterIndex == i {
					skipInterfaces = append(skipInterfaces, l.Attrs().Name)
				}
			}
		}
	}

	// query interfaces used for vlanConfigs and add them to list of skipped interfaces
	vlanConfigNICS, err := identifyClusterNetworks(nodeName, nodeCache, vlanConfigCache)
	if err != nil {
		return nil, err
	}

	skipInterfaces = append(skipInterfaces, vlanConfigNICS...)

	logrus.Debugf("skipping interfaces %v", skipInterfaces)

	// pciAddressess contains the pci addresses for the management nics
	var pciAddresses []string
	nics, err := ghw.Network()
	if err != nil {
		return nil, fmt.Errorf("error listing network info: %v", err)
	}

	for _, v := range skipInterfaces {
		for _, nic := range nics.NICs {
			if nic.Name == v {
				pciAddresses = append(pciAddresses, *nic.PCIAddress)
			}
		}
	}

	logrus.Debugf("skipping interfaces with pciAddresses: %v", pciAddresses)
	return pciAddresses, nil
}

// identifyClusterNetworks will identify vlanConfigs covering the current node and identify NICs in use for
// vlanconfigs
func identifyClusterNetworks(nodeName string, nodeCache ctlcorev1.NodeCache, vlanConfigCache ctlnetworkv1beta1.VlanConfigCache) ([]string, error) {
	var nicsList []string
	vlanConfigList, err := vlanConfigCache.List(labels.NewSelector())
	if err != nil {
		return nil, fmt.Errorf("error fetching vlanconfigs: %v", err)
	}
	for _, v := range vlanConfigList {
		managedNodes, found := v.Annotations[matchedNodesAnnotation]
		if !found { // if annotation not found, ignore as controller keeps checking on regular intervals
			continue
		}
		ok, err := currentNodeMatchesSelector(nodeName, managedNodes)
		if err != nil {
			return nil, fmt.Errorf("error evaulating nodes from selector: %v", err)
		}
		if ok {
			nicsList = append(nicsList, v.Spec.Uplink.NICs...)
		}
	}
	return nicsList, nil
}

// currentNodeMatchesSelector will use the label selectors from VlanConfig to identify if node is
// in the matching the VlanConfig
func currentNodeMatchesSelector(nodeName string, managedNodes string) (bool, error) {
	nodeNames := []string{}
	err := json.Unmarshal([]byte(managedNodes), &nodeNames)
	if err != nil {
		return false, fmt.Errorf("error unmarshalling matched-nodes: %v", err)
	}

	for _, v := range nodeNames {
		if v == nodeName {
			return true, nil
		}
	}
	return false, nil
}

type Node struct {
	Class       string `json:"class"`
	Id          string `json:"id"`
	BusInfo     string `json:"businfo"`
	LogicalName string `json:"logicalname"`
	Config      struct {
		Children []Node `json:"children"`
	} `json:"configuration"`
}

type DeviceInfo struct {
	PCIAddress       string
	NetworkInterface string
	LinkState        string // "UP", "DOWN"
}

func GetPCIEthernetDevicesWithStates() ([]DeviceInfo, error) {
	cmd := exec.Command("lshw", "-class", "network", "-json")
	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	var lshwData []Node
	err = json.Unmarshal(out, &lshwData)
	if err != nil {
		return nil, err
	}

	var devices []DeviceInfo
	findPCIDevices(lshwData, &devices)
	return devices, nil
}

func findPCIDevices(nodes []Node, devices *[]DeviceInfo) {
	for _, node := range nodes {
		if node.Class == "network" {
			pciAddress := ""
			networkInterface := ""
			if strings.HasPrefix(node.BusInfo, "pci") {
				pciAddress = strings.TrimPrefix(node.BusInfo, "pci:")
				networkInterface = node.LogicalName
				linkState, err := getLinkState(networkInterface)

				if err != nil {
					fmt.Printf("error getting link state: %s\n", err)
				}
				*devices = append(*devices, DeviceInfo{
					PCIAddress:       pciAddress,
					NetworkInterface: networkInterface,
					LinkState:        linkState,
				})
			}
		}
	}
}

func getLinkState(networkInterface string) (string, error) {
	linkStatePath := fmt.Sprintf("/sys/class/net/%s/operstate", networkInterface)
	content, err := ioutil.ReadFile(linkStatePath)
	if err != nil {
		return "", err
	}

	linkState := strings.TrimSpace(string(content))
	return linkState, nil
}
