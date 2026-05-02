package bandwidthscore

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// makeNodeWithAnnotation: 어노테이션 있는 노드 생성
func makeNodeWithAnnotation(name, bwUsage string) fwk.NodeInfo {
	ni := framework.NewNodeInfo()
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if bwUsage != "" {
		node.Annotations = map[string]string{
			BandwidthUsageAnnotation: bwUsage,
		}
	}
	ni.SetNode(node)
	return ni
}

func TestBandwidthScore(t *testing.T) {
	tests := []struct {
		name      string
		bwUsage   string
		wantScore int64
	}{
		{
			name:      "어노테이션 없음 → 50점",
			bwUsage:   "",
			wantScore: 50,
		},
		{
			name:      "BW 사용률 0% → 100점",
			bwUsage:   "0.0",
			wantScore: 100,
		},
		{
			name:      "BW 사용률 60% → 40점",
			bwUsage:   "0.6",
			wantScore: 40,
		},
		{
			name:      "BW 사용률 80% 임계치 경계 → 0점",
			bwUsage:   "0.9",
			wantScore: 0,
		},
	}

	pl := &BandwidthPlugin{
		bwThreshold: DefaultBwThreshold,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeInfo := makeNodeWithAnnotation("test-node", tt.bwUsage)
			score, status := pl.Score(
				context.Background(),
				nil,
				&v1.Pod{},
				nodeInfo,
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
