package kubernetes_test

import (
	"reflect"
	"testing"

	"github.com/weaveworks/scope/probe/kubernetes"
	"github.com/weaveworks/scope/report"
	"github.com/weaveworks/scope/test"
)

type mockRegistry struct {
	podsByID     map[string]kubernetes.Pod
	servicesByID map[string]kubernetes.Service
}

func (r *mockRegistry) Stop() {
}

func (r *mockRegistry) WalkPods(f func(p kubernetes.Pod)) {
	for _, p := range r.podsByID {
		f(p)
	}
}

func (r *mockRegistry) WalkServices(f func(s kubernetes.Service)) {
	for _, s := range r.servicesByID {
		f(s)
	}
}

var (
	mockPod1             = &mockPod{pod1}
	mockRegistryInstance = &mockRegistry{
		podsByID: map[string]kubernetes.Pod{
			mockPod1.ID(): mockPod1,
		},
		servicesByID: map[string]kubernetes.Service{},
	}
)

func TestReporter(t *testing.T) {
	want := report.MakeReport()
	want.Pod = report.Topology{
		Adjacency:     report.Adjacency{},
		EdgeMetadatas: report.EdgeMetadatas{},
		NodeMetadatas: report.NodeMetadatas{
			report.MakePodNodeID("ping", "pong"): report.MakeNodeMetadataWith(map[string]string{
				kubernetes.PodID:           "ping/pong",
				kubernetes.PodName:         "pong",
				kubernetes.Namespace:       "ping",
				kubernetes.PodCreated:      mockPod1.Created(),
				kubernetes.PodContainerIDs: "container1, container2",
			}),
		},
	}

	reporter := kubernetes.NewReporter(mockRegistryInstance)
	have, _ := reporter.Report()
	if !reflect.DeepEqual(want, have) {
		t.Errorf("%s", test.Diff(want, have))
	}
}
