package predictiveenqueue

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	fwk "k8s.io/kube-scheduler/framework"
)

const Name = "PredictiveEnqueue"

// PredictiveEnqueue는 방송 예약 시간 기반으로
// 스케줄링 큐 진입 시점을 제어
// 방송 시작 전 미리 리소스 확보
type PredictiveEnqueue struct {
	logger klog.Logger
}

var _ fwk.PreEnqueuePlugin = &PredictiveEnqueue{}

const (
	// Pod 어노테이션: 방송 예정 시작 시간 (RFC3339)
	// 예시: chijik.io/broadcast-time: "2024-01-01T15:00:00Z"
	BroadcastTimeAnnotation = "chijik.io/broadcast-time"

	// 방송 시작 몇 분 전부터 큐에 올릴지
	PrewarmDuration = 5 * time.Minute
)

func New(ctx context.Context, _ runtime.Object, _ fwk.Handle) (fwk.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &PredictiveEnqueue{
		logger: logger,
	}, nil
}

func (pl *PredictiveEnqueue) Name() string {
	return Name
}

// PreEnqueue: 방송 시작 5분 전 이전이면 큐 진입 차단
// 5분 전 이후가 되면 통과 → 스케줄러가 노드 배치 시작
func (pl *PredictiveEnqueue) PreEnqueue(
	ctx context.Context,
	pod *v1.Pod,
) *fwk.Status {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "PreEnqueue")

	broadcastTimeStr, ok := pod.Annotations[BroadcastTimeAnnotation]
	if !ok {
		// 어노테이션 없으면 일반 Pod처럼 바로 큐 진입
		return nil
	}

	broadcastTime, err := time.Parse(time.RFC3339, broadcastTimeStr)
	if err != nil {
		logger.V(5).Info("Invalid broadcast time annotation, skipping",
			"pod", pod.Name, "annotation", broadcastTimeStr)
		return nil
	}

	now := time.Now()
	prewarmTime := broadcastTime.Add(-PrewarmDuration)

	if now.Before(prewarmTime) {
		// 아직 5분 전이 안 됐음 → 큐 진입 차단
		logger.V(5).Info("Pod not yet ready to enqueue",
			"pod", pod.Name,
			"broadcast-time", broadcastTime,
			"enqueue-after", prewarmTime)
		return fwk.NewStatus(fwk.UnschedulableAndUnresolvable,
			"waiting for broadcast prewarm window")
	}

	logger.V(5).Info("Pod entering prewarm window, enqueuing",
		"pod", pod.Name,
		"broadcast-time", broadcastTime)
	return nil
}
