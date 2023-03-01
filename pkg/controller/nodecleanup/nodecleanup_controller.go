package nodecleanup

import (
	"context"
	"fmt"

	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/harvester/pcidevices/pkg/generated/controllers/devices.harvesterhci.io/v1beta1"
)

type Handler struct {
	ctx        context.Context
	pdcClient  v1beta1.PCIDeviceClaimClient
	pdClient   v1beta1.PCIDeviceClient
	nodeClient corecontrollers.NodeController
	clientSet  *kubernetes.Clientset
}

func Register(
	ctx context.Context,
	pdcClient v1beta1.PCIDeviceClaimController,
	pdClient v1beta1.PCIDeviceController,
	nodeClient corecontrollers.NodeController,
	clientSet *kubernetes.Clientset,
) error {
	handler := &Handler{
		ctx:        ctx,
		pdcClient:  pdcClient,
		pdClient:   pdClient,
		nodeClient: nodeClient,
		clientSet:  clientSet,
	}
	logrus.Info("Registering Node cleanup controller")
	nodeClient.OnRemove(ctx, "node-remove", handler.OnRemove)
	return nil
}

func (h *Handler) OnRemove(_ string, node *v1.Node) (*v1.Node, error) {
	if node == nil || node.DeletionTimestamp == nil {
		return node, nil
	}
	logrus.Infof("[node=%s]OnRemove", node.Name)
	// Delete all of that Node's PCIDeviceClaims
	pdcs, err := h.pdcClient.List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("error getting pdcs: %s", err)
		return node, err
	}
	for _, pdc := range pdcs.Items {
		if pdc.Spec.NodeName != node.Name {
			continue
		}
		err = h.pdcClient.Delete(pdc.Name, &metav1.DeleteOptions{})
		if err != nil {
			logrus.Errorf("error deleting pdc: %s", err)
			return node, err
		}
	}
	// Delete all of that Node's PCIDevices
	selector := fmt.Sprintf("nodename=%s", node.Name)
	pds, err := h.pdClient.List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		logrus.Errorf("error getting pds: %s", err)
		return node, err
	}
	for _, pd := range pds.Items {
		err = h.pdClient.Delete(pd.Name, &metav1.DeleteOptions{})
		if err != nil {
			logrus.Errorf("error deleting pd: %s", err)
			return node, err
		}
	}
	return node, nil
}
