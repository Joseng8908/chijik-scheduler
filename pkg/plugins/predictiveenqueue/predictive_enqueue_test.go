package predictiveenqueue

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fwk "k8s.io/kube-scheduler/framework"
)

func makePodWithBroadcastTime(broadcastTime time.Time) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Annotations: map[string]string{
				BroadcastTimeAnnotation: broadcastTime.Format(time.RFC3339),
			},
		},
	}
}

func TestPredictiveEnqueue(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		pod      *v1.Pod
		wantCode fwk.Code
	}{
		{
			name:     "어노테이션 없음 → 바로 큐 진입",
			pod:      &v1.Pod{},
			wantCode: fwk.Success,
		},
		{
			name:     "방송 시작 1시간 후 → 아직 프리웜 윈도우 아님 → 차단",
			pod:      makePodWithBroadcastTime(now.Add(1 * time.Hour)),
			wantCode: fwk.UnschedulableAndUnresolvable,
		},
		{
			name:     "방송 시작 3분 후 → 프리웜 윈도우 진입 → 통과",
			pod:      makePodWithBroadcastTime(now.Add(3 * time.Minute)),
			wantCode: fwk.Success,
		},
		{
			name:     "방송 이미 시작 → 통과",
			pod:      makePodWithBroadcastTime(now.Add(-10 * time.Minute)),
			wantCode: fwk.Success,
		},
		{
			name: "잘못된 시간 형식 → 통과 (무시)",
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						BroadcastTimeAnnotation: "invalid-time",
					},
				},
			},
			wantCode: fwk.Success,
		},
	}

	pl := &PredictiveEnqueue{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := pl.PreEnqueue(context.Background(), tt.pod)

			if tt.wantCode == fwk.Success {
				if status != nil && status.Code() != fwk.Success {
					t.Errorf("PreEnqueue() status = %v, want Success", status)
				}
			} else {
				if status == nil || status.Code() != tt.wantCode {
					t.Errorf("PreEnqueue() status = %v, want %v", status, tt.wantCode)
				}
			}
		})
	}
}
