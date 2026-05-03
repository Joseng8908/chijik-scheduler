package bandwidthscore

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

func makeNodeWithAnnotation(name, bwUsage string) *framework.NodeInfo {
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
			name:      "BW 사용률 90% → 임계치 초과 → 0점",
			bwUsage:   "0.9",
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			nodeInfo := makeNodeWithAnnotation("test-node", tt.bwUsage)
			fakeSharedLister := &fakeSharedLister{nodes: []*framework.NodeInfo{nodeInfo}}

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

			score, status := pl.(*BandwidthPlugin).Score(
				ctx, nil, &v1.Pod{}, "test-node",
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
