package predictiveenqueue

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const Name = "PredictiveEnqueue"

type PredictiveEnqueue struct {
	logger klog.Logger
}

var _ framework.PreEnqueuePlugin = &PredictiveEnqueue{}

const (
	BroadcastTimeAnnotation = "chijik.io/broadcast-time"
	PrewarmDuration         = 5 * time.Minute
)

func New(ctx context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	logger := klog.FromContext(ctx).WithValues("plugin", Name)
	return &PredictiveEnqueue{
		logger: logger,
	}, nil
}

func (pl *PredictiveEnqueue) Name() string {
	return Name
}

func (pl *PredictiveEnqueue) PreEnqueue(
	ctx context.Context,
	pod *v1.Pod,
) *framework.Status {
	logger := klog.FromContext(klog.NewContext(ctx, pl.logger)).WithValues("ExtensionPoint", "PreEnqueue")

	broadcastTimeStr, ok := pod.Annotations[BroadcastTimeAnnotation]
	if !ok {
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
		logger.V(5).Info("Pod not yet ready to enqueue",
			"pod", pod.Name,
			"broadcast-time", broadcastTime,
			"enqueue-after", prewarmTime)
		return framework.NewStatus(framework.UnschedulableAndUnresolvable,
			"waiting for broadcast prewarm window")
	}

	logger.V(5).Info("Pod entering prewarm window, enqueuing",
		"pod", pod.Name,
		"broadcast-time", broadcastTime)
	return nil
}
