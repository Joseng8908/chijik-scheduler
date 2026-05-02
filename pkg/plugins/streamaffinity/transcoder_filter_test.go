package streamaffinity

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func makeNode(name string) fwk.NodeInfo {
	ni := framework.NewNodeInfo()
	ni.SetNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
	return ni
}

func makeNodeWithIngestPod(name, streamID string) fwk.NodeInfo {
	ni := framework.NewNodeInfo()
	ni.SetNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
	ni.AddPod(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingest-pod",
			Labels: map[string]string{
				WorkloadLabel: IngestWorkload,
				StreamIDLabel: streamID,
			},
		},
	})
	return ni
}

func makeTranscoderPod(streamID string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "transcoder-pod",
			Labels: map[string]string{
				WorkloadLabel: "transcoder",
				StreamIDLabel: streamID,
			},
		},
	}
}

func TestStreamAffinity(t *testing.T) {
	tests := []struct {
		name      string
		pod       *v1.Pod
		nodeInfo  fwk.NodeInfo
		wantScore int64
	}{
		{
			name:      "transcoder 아님 → 0점",
			pod:       &v1.Pod{},
			nodeInfo:  makeNode("test-node"),
			wantScore: 0,
		},
		{
			name:      "같은 stream-id Ingest Pod 없음 → 0점",
			pod:       makeTranscoderPod("streamer-a"),
			nodeInfo:  makeNode("test-node"),
			wantScore: 0,
		},
		{
			name:      "같은 stream-id Ingest Pod 있음 → 100점",
			pod:       makeTranscoderPod("streamer-a"),
			nodeInfo:  makeNodeWithIngestPod("test-node", "streamer-a"),
			wantScore: fwk.MaxNodeScore,
		},
		{
			name:      "다른 stream-id Ingest Pod → 0점",
			pod:       makeTranscoderPod("streamer-a"),
			nodeInfo:  makeNodeWithIngestPod("test-node", "streamer-b"),
			wantScore: 0,
		},
	}

	pl := &StreamAffinity{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, status := pl.Score(
				context.Background(),
				nil,
				tt.pod,
				tt.nodeInfo,
			)
			if status != nil && status.Code() != fwk.Success {
				t.Errorf("Score() status = %v", status)
			}
			if score != tt.wantScore {
				t.Errorf("Score() = %v, want %v", score, tt.wantScore)
			}
		})
	}
}
