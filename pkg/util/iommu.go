package iommu

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func GroupMapForPCIDevices(groupPaths []string) map[string]int {
	groupMap := make(map[string]int)
	for _, groupPath := range groupPaths {
		address := strings.Split(groupPath, "/")[6]
		group := GroupForPCIDevice(address, groupPaths)
		groupMap[address] = group
	}
	return groupMap
}

// return the iommu group of the PCI device
func GroupForPCIDevice(address string, groupPaths []string) int {
	for _, groupPath := range groupPaths {
		if strings.HasSuffix(groupPath, address) {
			// extract iommu group from path and return
			iommuGroupStr := strings.Split(groupPath, "/")[4]
			iommuGroup, err := strconv.Atoi(iommuGroupStr)
			if err != nil {
				// TODO log error
				return -1
			}
			return iommuGroup
		}
	}
	return 0
}

// return all paths like /sys/kernel/iommu_groups/$GROUP/devices/$DEVICE
func GroupPaths() []string {
	// list all iommu groups
	iommuGroups, err := ioutil.ReadDir("/sys/kernel/iommu_groups")
	if err != nil {
		// TODO log the error
		return []string{}
	}
	var groupPaths []string = []string{}
	for _, group := range iommuGroups {
		path := fmt.Sprintf("/sys/kernel/iommu_groups/%s/devices", group.Name())
		devices, err := ioutil.ReadDir(path)
		if err != nil {
			// TODO log the error
			continue
		}
		for _, device := range devices {
			groupPath := fmt.Sprintf("/sys/kernel/iommu_groups/%s/devices/%s", group.Name(), device.Name())
			groupPaths = append(groupPaths, groupPath)
		}
	}
	return groupPaths
}
