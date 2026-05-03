package bandwidthscore

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "BandwidthScore"

type BandwidthPlugin struct {
	logger      klog.Logger
	handle      framework.Handle
	bwThreshold float64
}

var _ framework.ScorePlugin = &BandwidthPlugin{}

const (
	BandwidthUsageAnnotation = "chijik.io/bandwidth-usage"
	DefaultBwThreshold       = 0.8
)

func New(ctx context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &BandwidthPlugin{
		logger:      logger,
		handle:      h,
		bwThreshold: DefaultBwThreshold,
	}, nil
}

func (pl *BandwidthPlugin) Name() string {
	return Name
}

func (pl *BandwidthPlugin) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Score")

	nodeInfo, err := pl.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error,
			fmt.Sprintf("getting node %q: %v", nodeName, err))
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node not found")
	}

	bwUsageStr, ok := node.Annotations[BandwidthUsageAnnotation]
	if !ok {
		logger.V(5).Info("No bandwidth annotation found, using default score",
			"node", node.Name)
		return 50, nil
	}

	bwUsage, err := strconv.ParseFloat(bwUsageStr, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error,
			fmt.Sprintf("invalid bandwidth annotation on node %s: %v", node.Name, err))
	}

	if bwUsage > pl.bwThreshold {
		logger.V(5).Info("Node BW usage exceeds threshold, score=0",
			"node", node.Name, "usage", bwUsage, "threshold", pl.bwThreshold)
		return 0, nil
	}

	score := int64((1.0 - bwUsage) * float64(framework.MaxNodeScore))
	logger.V(5).Info("Node BW score calculated",
		"node", node.Name, "usage", bwUsage, "score", score)
	return score, nil
}

func (pl *BandwidthPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
