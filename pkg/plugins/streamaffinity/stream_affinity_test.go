package streamaffinity

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	clientsetfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/defaultbinder"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/queuesort"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	tf "k8s.io/kubernetes/pkg/scheduler/testing/framework"
)

var _ framework.SharedLister = &fakeSharedLister{}

type fakeSharedLister struct {
	nodes []*framework.NodeInfo
}

func (f *fakeSharedLister) StorageInfos() framework.StorageInfoLister {
	return nil
}

func (f *fakeSharedLister) NodeInfos() framework.NodeInfoLister {
	return tf.NodeInfoLister(f.nodes)
}

func makeNode(name string) *framework.NodeInfo {
	ni := framework.NewNodeInfo()
	ni.SetNode(&v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
	return ni
}

func makeNodeWithIngestPod(name, streamID string) *framework.NodeInfo {
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
		nodeInfo  *framework.NodeInfo
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
			wantScore: framework.MaxNodeScore,
		},
		{
			name:      "다른 stream-id Ingest Pod → 0점",
			pod:       makeTranscoderPod("streamer-a"),
			nodeInfo:  makeNodeWithIngestPod("test-node", "streamer-b"),
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			fakeSharedLister := &fakeSharedLister{nodes: []*framework.NodeInfo{tt.nodeInfo}}
			cs := clientsetfake.NewSimpleClientset()
			informerFactory := informers.NewSharedInformerFactory(cs, 0)

			registeredPlugins := []tf.RegisterPluginFunc{
				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
				tf.RegisterScorePlugin(Name, New, 1),
			}

			fh, err := tf.NewFramework(
				ctx,
				registeredPlugins,
				"default-scheduler",
				frameworkruntime.WithClientSet(cs),
				frameworkruntime.WithInformerFactory(informerFactory),
				frameworkruntime.WithSnapshotSharedLister(fakeSharedLister),
			)
			if err != nil {
				t.Fatalf("failed to create framework: %v", err)
			}

			pl, err := New(ctx, &runtime.Unknown{}, fh)
			if err != nil {
				t.Fatalf("failed to create plugin: %v", err)
			}

			score, status := pl.(*StreamAffinity).Score(
				ctx, nil, tt.pod, tt.nodeInfo.Node().Name,
			)
			if status != nil && !status.IsSuccess() {
				t.Errorf("Score() status = %v", status)
			}
			if score != tt.wantScore {
				t.Errorf("Score() = %v, want %v", score, tt.wantScore)
			}
		})
	}
}
