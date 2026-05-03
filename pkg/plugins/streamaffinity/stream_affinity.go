package streamaffinity

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "StreamAffinity"

type StreamAffinity struct {
	logger klog.Logger
	handle framework.Handle
}

var _ framework.ScorePlugin = &StreamAffinity{}

const (
	StreamIDLabel  = "chijik.io/stream-id"
	WorkloadLabel  = "chijik.io/workload"
	IngestWorkload = "ingest"
)

func New(ctx context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &StreamAffinity{
		logger: logger,
		handle: h,
	}, nil
}

func (pl *StreamAffinity) Name() string {
	return Name
}

func (pl *StreamAffinity) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	nodeName string,
) (int64, *framework.Status) {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Score")

	if pod.Labels[WorkloadLabel] != "transcoder" {
		return 0, nil
	}

	streamID, ok := pod.Labels[StreamIDLabel]
	if !ok {
		return 0, nil
	}

	nodeInfo, err := pl.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error,
			fmt.Sprintf("getting node %q: %v", nodeName, err))
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node not found")
	}

	for _, podInfo := range nodeInfo.Pods {
		p := podInfo.Pod
		if p.Labels[WorkloadLabel] == IngestWorkload &&
			p.Labels[StreamIDLabel] == streamID {
			logger.V(5).Info("Found matching Ingest pod on node",
				"node", node.Name, "stream-id", streamID)
			return framework.MaxNodeScore, nil
		}
	}

	return 0, nil
}

func (pl *StreamAffinity) ScoreExtensions() framework.ScoreExtensions {
	return nil
}
