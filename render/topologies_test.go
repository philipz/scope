package render_test

import (
	"reflect"
	"testing"

	"github.com/weaveworks/scope/probe/docker"
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
	have := expected.Sterilize(render.ContainerWithImageNameRenderer{}.Render(test.Report))
	want := expected.RenderedContainers
	if !reflect.DeepEqual(want, have) {
		t.Error(test.Diff(want, have))
	}
}

func TestContainerFilterRenderer(t *testing.T) {
	// tag on of the containers in the topology and ensure
	// it is filtered out correctly.
	input := test.Report.Copy()
	input.Container.Nodes[test.ClientContainerNodeID].Metadata[docker.LabelPrefix+"works.weave.role"] = "system"
	have := expected.Sterilize(render.FilterSystem(render.ContainerWithImageNameRenderer{}).Render(input))
	want := expected.RenderedContainers.Copy()
	delete(want, test.ClientContainerID)
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
