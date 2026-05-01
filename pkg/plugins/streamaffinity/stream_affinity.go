package streamaffinity

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	fwk "k8s.io/kube-scheduler/framework"
)

const Name = "StreamAffinity"

// StreamAffinity는 같은 stream-id를 가진 Ingest Pod와
// Transcoder Pod를 같은 노드에 배치하도록 점수 부여
type StreamAffinity struct {
	logger klog.Logger
	handle fwk.Handle
}

var _ fwk.ScorePlugin = &StreamAffinity{}

const (
	// Pod 레이블 키
	StreamIDLabel   = "chijik.io/stream-id"   // 스트림 식별자
	WorkloadLabel   = "chijik.io/workload"     // ingest or transcoder
	IngestWorkload  = "ingest"
)

func New(ctx context.Context, _ runtime.Object, h fwk.Handle) (fwk.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &StreamAffinity{
		logger: logger,
		handle: h,
	}, nil
}

func (pl *StreamAffinity) Name() string {
	return Name
}

// Score: 같은 stream-id의 Ingest Pod가 있는 노드에 높은 점수
func (pl *StreamAffinity) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo fwk.NodeInfo,
) (int64, *fwk.Status) {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Score")

	// Transcoder Pod만 적용
	if pod.Labels[WorkloadLabel] != "transcoder" {
		return 0, nil
	}

	streamID, ok := pod.Labels[StreamIDLabel]
	if !ok {
		return 0, nil
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, fwk.NewStatus(fwk.Error, "node not found")
	}

	// 해당 노드에 같은 stream-id의 Ingest Pod 있는지 확인
	for _, podInfo := range nodeInfo.GetPods() {
		p := podInfo.GetPod()
		if p.Labels[WorkloadLabel] == IngestWorkload &&
			p.Labels[StreamIDLabel] == streamID {
			logger.V(5).Info("Found matching Ingest pod on node",
				"node", node.Name, "stream-id", streamID)
			return fwk.MaxNodeScore, nil // 100점
		}
	}

	fmt.Sprintf("no matching ingest pod on node %s", node.Name)
	return 0, nil
}

func (pl *StreamAffinity) ScoreExtensions() fwk.ScoreExtensions {
	return nil
}
