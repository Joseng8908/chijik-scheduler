package transcoderfilter

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "TranscoderFilter"

type TranscoderFilter struct {
	logger      klog.Logger
	bwThreshold float64
}

var _ framework.FilterPlugin = &TranscoderFilter{}

const (
	BandwidthUsageAnnotation = "chijik.io/bandwidth-usage"
	WorkloadLabel            = "chijik.io/workload"
	DefaultBwThreshold       = 0.9
)

func New(ctx context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &TranscoderFilter{
		logger:      logger,
		bwThreshold: DefaultBwThreshold,
	}, nil
}

func (pl *TranscoderFilter) Name() string {
	return Name
}

func (pl *TranscoderFilter) Filter(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeInfo *framework.NodeInfo,
) *framework.Status {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Filter")

	if pod.Labels[WorkloadLabel] != "transcoder" {
		return nil
	}

	node := nodeInfo.Node()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}

	bwUsageStr, ok := node.Annotations[BandwidthUsageAnnotation]
	if !ok {
		return nil
	}

	bwUsage, err := strconv.ParseFloat(bwUsageStr, 64)
	if err != nil {
		return framework.NewStatus(framework.Error,
			fmt.Sprintf("invalid bandwidth annotation on node %s: %v", node.Name, err))
	}

	if bwUsage > pl.bwThreshold {
		logger.V(5).Info("Node filtered due to high BW usage",
			"node", node.Name, "usage", bwUsage, "threshold", pl.bwThreshold)
		return framework.NewStatus(framework.Unschedulable,
			fmt.Sprintf("node %s BW usage %.2f exceeds threshold %.2f",
				node.Name, bwUsage, pl.bwThreshold))
	}

	return nil
}
