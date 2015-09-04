package render_test

import (
	"reflect"
	"testing"

	"github.com/weaveworks/scope/render"
	"github.com/weaveworks/scope/render/expected"
	"github.com/weaveworks/scope/test"
)

func TestProcessRenderer(t *testing.T) {
	have := expected.Sterilize(render.ProcessRenderer.Render(test.Report))
	want := expected.RenderedProcesses
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestProcessNameRenderer(t *testing.T) {
	have := expected.Sterilize(render.ProcessNameRenderer.Render(test.Report))
	want := expected.RenderedProcessNames
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestContainerRenderer(t *testing.T) {
	have := expected.Sterilize(render.ContainerRenderer.Render(test.Report))
	want := expected.RenderedContainers
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestContainerImageRenderer(t *testing.T) {
	have := expected.Sterilize(render.ContainerImageRenderer.Render(test.Report))
	want := expected.RenderedContainerImages
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestHostRenderer(t *testing.T) {
	have := expected.Sterilize(render.HostRenderer.Render(test.Report))
	want := expected.RenderedHosts
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestKubernetesRenderer(t *testing.T) {
	have := expected.Sterilize(render.KubernetesRenderer.Render(test.Report))
	want := expected.RenderedKubernetes
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}
