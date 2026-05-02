package transcoderfilter

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fwk "k8s.io/kube-scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

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

func makePod(workload string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}
	if workload != "" {
		pod.Labels = map[string]string{
			WorkloadLabel: workload,
		}
	}
	return pod
}

func TestTranscoderFilter(t *testing.T) {
	tests := []struct {
		name       string
		workload   string
		bwUsage    string
		wantCode   fwk.Code
	}{
		{
			name:     "transcoder 아님 → 통과",
			workload: "ingest",
			bwUsage:  "0.95",
			wantCode: fwk.Success,
		},
		{
			name:     "transcoder + 어노테이션 없음 → 통과",
			workload: "transcoder",
			bwUsage:  "",
			wantCode: fwk.Success,
		},
		{
			name:     "transcoder + BW 60% → 통과",
			workload: "transcoder",
			bwUsage:  "0.6",
			wantCode: fwk.Success,
		},
		{
			name:     "transcoder + BW 95% → 임계치 초과 → 차단",
			workload: "transcoder",
			bwUsage:  "0.95",
			wantCode: fwk.Unschedulable,
		},
	}

	pl := &TranscoderFilter{
		bwThreshold: DefaultBwThreshold,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := makePod(tt.workload)
			nodeInfo := makeNodeWithAnnotation("test-node", tt.bwUsage)

			status := pl.Filter(
				context.Background(),
				nil,
				pod,
				nodeInfo,
			)

			if tt.wantCode == fwk.Success {
				if status != nil && status.Code() != fwk.Success {
					t.Errorf("Filter() status = %v, want Success", status)
				}
			} else {
				if status == nil || status.Code() != tt.wantCode {
					t.Errorf("Filter() status = %v, want %v", status, tt.wantCode)
				}
			}
		})
	}
}
