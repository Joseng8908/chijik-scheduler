package transcoderfilter

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	fwk "k8s.io/kube-scheduler/framework"
)

const Name = "TranscoderFilter"

// TranscoderFilter는 BW 임계치 초과 노드를 필터링
// BandwidthScore가 점수를 낮추는 것과 달리
// 여기서는 임계치 초과 노드를 아예 제거
type TranscoderFilter struct {
	logger      klog.Logger
	bwThreshold float64
}

var _ fwk.FilterPlugin = &TranscoderFilter{}

const (
	BandwidthUsageAnnotation = "chijik.io/bandwidth-usage"
	DefaultBwThreshold       = 0.9 // Filter는 90%로 더 관대하게
	WorkloadLabel="chijik.io/workload"
)

func New(ctx context.Context, _ runtime.Object, h fwk.Handle) (fwk.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &TranscoderFilter{
		logger:      logger,
		bwThreshold: DefaultBwThreshold,
	}, nil
}

func (pl *TranscoderFilter) Name() string {
	return Name
}

// Filter: BW 사용률이 임계치 초과하면 노드 제거
// BandwidthScore와 역할 분담:
//   TranscoderFilter → 90% 초과 노드 완전 제거
//   BandwidthScore   → 남은 노드 중 BW 잔량으로 순위 결정
func (pl *TranscoderFilter) Filter(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo fwk.NodeInfo,
) *fwk.Status {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Filter")

	// 트랜스코딩 Pod만 필터링 적용
	// 레이블: chijik.io/workload=transcoder
	if pod.Labels[WorkloadLabel] != "transcoder" {
		return nil // 트랜스코딩 Pod 아니면 필터 통과
	}

	node := nodeInfo.Node()
	if node == nil {
		return fwk.NewStatus(fwk.Error, "node not found")
	}

	bwUsageStr, ok := node.Annotations[BandwidthUsageAnnotation]
	if !ok {
		// 어노테이션 없으면 통과 (알 수 없으면 허용)
		return nil
	}

	bwUsage, err := strconv.ParseFloat(bwUsageStr, 64)
	if err != nil {
		return fwk.NewStatus(fwk.Error,
			fmt.Sprintf("invalid bandwidth annotation on node %s: %v", node.Name, err))
	}

	if bwUsage > pl.bwThreshold {
		logger.V(5).Info("Node filtered due to high BW usage",
			"node", node.Name, "usage", bwUsage, "threshold", pl.bwThreshold)
		return fwk.NewStatus(fwk.Unschedulable,
			fmt.Sprintf("node %s BW usage %.2f exceeds threshold %.2f",
				node.Name, bwUsage, pl.bwThreshold))
	}

	return nil
}
