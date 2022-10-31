package iommu

import "testing"

func TestGroupForPCIDevice(t *testing.T) {
	type args struct {
		iommuGroups []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Get IOMMU group for 0000:3e:04.2",
			args: args{
				iommuGroups: []string{
					"/sys/kernel/iommu_groups/9/devices/0000:00:1c.0",
					"/sys/kernel/iommu_groups/9/devices/0000:00:1c.5",
					"/sys/kernel/iommu_groups/9/devices/0000:06:00.0",
					"/sys/kernel/iommu_groups/9/devices/0000:05:00.0",
					"/sys/kernel/iommu_groups/27/devices/0000:3e:04.2",
					"/sys/kernel/iommu_groups/27/devices/0000:3e:04.0",
					"/sys/kernel/iommu_groups/27/devices/0000:3e:04.3",
					"/sys/kernel/iommu_groups/27/devices/0000:3e:04.1",
				},
			},
			want: 27,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := "0000:3e:04.2"
			if got := GroupForPCIDevice(address, tt.args.iommuGroups); got != tt.want {
				t.Errorf("GroupForPCIDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
