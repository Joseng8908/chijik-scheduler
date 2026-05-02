package bandwidthscore

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	fwk "k8s.io/kube-scheduler/framework"
)

const Name = "BandwidthScore"

// BandwidthPlugin은 노드의 네트워크 BW 잔량 기반으로 점수를 계산
// 치지직 트랜스코딩 워크로드 최적 배치 목적
type BandwidthPlugin struct {
	logger      klog.Logger
	handle      fwk.Handle
	bwThreshold float64 // BW 사용률 임계치 (0.0~1.0)
}

var _ fwk.ScorePlugin = &BandwidthPlugin{}

const (
	// 노드 어노테이션 키 (추후 Prometheus 쿼리로 교체)
	// 형식: chijik.io/bandwidth-usage: "0.6"
	BandwidthUsageAnnotation = "chijik.io/bandwidth-usage"

	DefaultBwThreshold = 0.8 // 기본 임계치 80%
)

func New(ctx context.Context, _ runtime.Object, h fwk.Handle) (fwk.Plugin, error) {
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

// Score: 노드별 네트워크 BW 잔량 기반 점수 계산
// BW 사용률 낮을수록 높은 점수
// 임계치(80%) 초과 노드 = 0점
func (pl *BandwidthPlugin) Score(
	ctx context.Context,
	state fwk.CycleState,
	pod *v1.Pod,
	nodeInfo fwk.NodeInfo,
) (int64, *fwk.Status) {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "Score")

	node := nodeInfo.Node()
	if node == nil {
		return 0, fwk.NewStatus(fwk.Error, "node not found")
	}

	// 노드 어노테이션에서 BW 사용률 읽기
	bwUsageStr, ok := node.Annotations[BandwidthUsageAnnotation]
	if !ok {
		// 어노테이션 없으면 중간 점수 반환
		logger.V(5).Info("No bandwidth annotation found, using default score",
			"node", node.Name)
		return 50, nil
	}

	bwUsage, err := strconv.ParseFloat(bwUsageStr, 64)
	if err != nil {
		return 0, fwk.NewStatus(fwk.Error,
			fmt.Sprintf("invalid bandwidth usage annotation on node %s: %v", node.Name, err))
	}

	if bwUsage > pl.bwThreshold {
		logger.V(5).Info("Node BW usage exceeds threshold, score=0",
			"node", node.Name, "usage", bwUsage, "threshold", pl.bwThreshold)
		return 0, nil
	}

	score := int64((1.0 - bwUsage) * float64(fwk.MaxNodeScore))
	logger.V(5).Info("Node BW score calculated",
		"node", node.Name, "usage", bwUsage, "score", score)

	return score, nil
}

func (pl *BandwidthPlugin) ScoreExtensions() fwk.ScoreExtensions {
	return nil
}
