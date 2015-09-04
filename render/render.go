package render

import (
	"log"
	"strings"

	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/report"
)

// Renderer is something that can render a report to a set of RenderableNodes.
type Renderer interface {
	Render(report.Report) RenderableNodes
	EdgeMetadata(rpt report.Report, localID, remoteID string) report.EdgeMetadata
}

// Reduce renderer is a Renderer which merges together the output of several
// other renderers.
type Reduce []Renderer

// MakeReduce is the only sane way to produce a Reduce Renderer.
func MakeReduce(renderers ...Renderer) Renderer {
	return Reduce(renderers)
}

// Render produces a set of RenderableNodes given a Report.
func (r Reduce) Render(rpt report.Report) RenderableNodes {
	result := RenderableNodes{}
	for _, renderer := range r {
		result = result.Merge(renderer.Render(rpt))
	}
	return result
}

// EdgeMetadata produces an EdgeMetadata for a given edge.
func (r Reduce) EdgeMetadata(rpt report.Report, localID, remoteID string) report.EdgeMetadata {
	metadata := report.EdgeMetadata{}
	for _, renderer := range r {
		metadata = metadata.Merge(renderer.EdgeMetadata(rpt, localID, remoteID))
	}
	return metadata
}

// Map is a Renderer which produces a set of RenderableNodes from the set of
// RenderableNodes produced by another Renderer.
type Map struct {
	MapFunc
	Renderer
}

// Render transforms a set of RenderableNodes produces by another Renderer.
// using a map function
func (m Map) Render(rpt report.Report) RenderableNodes {
	output, _ := m.render(rpt)
	return output
}

func (m Map) render(rpt report.Report) (RenderableNodes, map[string]report.IDList) {
	var (
		input         = m.Renderer.Render(rpt)
		output        = RenderableNodes{}
		mapped        = map[string]report.IDList{} // input node ID -> output node IDs
		adjacencies   = map[string]report.IDList{} // output node ID -> input node Adjacencies
		localNetworks = LocalNetworks(rpt)
	)

	// Rewrite all the nodes according to the map function
	for _, inRenderable := range input {
		for _, outRenderable := range m.MapFunc(inRenderable, localNetworks) {
			existing, ok := output[outRenderable.ID]
			if ok {
				outRenderable = outRenderable.Merge(existing)
			}

			output[outRenderable.ID] = outRenderable
			mapped[inRenderable.ID] = mapped[inRenderable.ID].Add(outRenderable.ID)
			adjacencies[outRenderable.ID] = adjacencies[outRenderable.ID].Merge(inRenderable.Adjacency)
		}
	}

	// Rewrite Adjacency for new node IDs.
	for outNodeID, inAdjacency := range adjacencies {
		outAdjacency := report.MakeIDList()
		for _, inAdjacent := range inAdjacency {
			for _, outAdjacent := range mapped[inAdjacent] {
				outAdjacency = outAdjacency.Add(outAdjacent)
			}
		}
		outNode := output[outNodeID]
		outNode.Adjacency = outAdjacency
		output[outNodeID] = outNode
	}

	return output, mapped
}

// EdgeMetadata gives the metadata of an edge from the perspective of the
// srcRenderableID. Since an edgeID can have multiple edges on the address
// level, it uses the supplied mapping function to translate address IDs to
// renderable node (mapped) IDs.
func (m Map) EdgeMetadata(rpt report.Report, srcRenderableID, dstRenderableID string) report.EdgeMetadata {
	// First we need to map the ids in this layer into the ids in the underlying layer
	_, mapped := m.render(rpt)        // this maps from old -> new
	inverted := map[string][]string{} // this maps from new -> old(s)
	for k, vs := range mapped {
		for _, v := range vs {
			existing := inverted[v]
			existing = append(existing, k)
			inverted[v] = existing
		}
	}

	// Now work out a slice of edges this edge is constructed from
	oldEdges := []struct{ src, dst string }{}
	for _, oldSrcID := range inverted[srcRenderableID] {
		for _, oldDstID := range inverted[dstRenderableID] {
			oldEdges = append(oldEdges, struct{ src, dst string }{oldSrcID, oldDstID})
		}
	}

	// Now recurse for each old edge
	output := report.EdgeMetadata{}
	for _, edge := range oldEdges {
		metadata := m.Renderer.EdgeMetadata(rpt, edge.src, edge.dst)
		output = output.Merge(metadata)
	}
	return output
}

// CustomRenderer allow for mapping functions that recived the entire topology
// in one call - useful for functions that need to consider the entire graph
type CustomRenderer struct {
	RenderFunc func(RenderableNodes) RenderableNodes
	Renderer
}

// Render implements Renderer
func (c CustomRenderer) Render(rpt report.Report) RenderableNodes {
	return c.RenderFunc(c.Renderer.Render(rpt))
}

// IsConnected is the key added to Node.Metadata by ColorConnected
// to indicate a node has an edge pointing to it or from it
const IsConnected = "is_connected"

// OnlyConnected filters out unconnected RenderedNodes
func OnlyConnected(input RenderableNodes) RenderableNodes {
	output := RenderableNodes{}
	for id, node := range ColorConnected(input) {
		if _, ok := node.Node.Metadata[IsConnected]; ok {
			output[id] = node
		}
	}
	return output
}

// FilterUnconnected produces a renderer that filters unconnected nodes
// from the given renderer
func FilterUnconnected(r Renderer) Renderer {
	return CustomRenderer{
		RenderFunc: OnlyConnected,
		Renderer:   r,
	}
}

// ColorConnected colors nodes with the IsConnected key if
// they have edges to or from them.
func ColorConnected(input RenderableNodes) RenderableNodes {
	connected := map[string]struct{}{}
	void := struct{}{}

	for id, node := range input {
		if len(node.Adjacency) == 0 {
			continue
		}

		connected[id] = void
		for _, id := range node.Adjacency {
			connected[id] = void
		}
	}

	for id := range connected {
		node := input[id]
		node.Node.Metadata[IsConnected] = "true"
		input[id] = node
	}
	return input
}

// Filter removes nodes from a view based on a predicate.
type Filter struct {
	Renderer
	f func(RenderableNode) bool
}

// Render implements Renderer
func (f Filter) Render(rpt report.Report) RenderableNodes {
	output := RenderableNodes{}
	for id, node := range f.Renderer.Render(rpt) {
		if f.f(node) {
			output[id] = node
		}
	}
	return output
}

// FilterSystem is a Renderer which filters out system nodes.
func FilterSystem(r Renderer) Renderer {
	return CustomRenderer{
		RenderFunc: nonSystem,
		Renderer:   r,
	}
}

func nonSystem(input RenderableNodes) RenderableNodes {
	// Deleted nodes also need to be cut as destinations in adjacency lists.
	// So, we need to count all nodes to reject, before effecting mutation.
	reject := map[string]struct{}{}
	for id, node := range input {
		if isSystem(node.Metadata) {
			reject[id] = struct{}{}
		}
	}

	output := RenderableNodes{}
	for id, node := range input {
		// Check for completely rejected node.
		if _, ok := reject[id]; ok {
			continue
		}

		// Accepted node, but maybe some bad adjacencies.
		// Cut those before continuing.
		newAdjacency := make(report.IDList, 0, len(node.Adjacency))
		for _, dstID := range node.Adjacency {
			if _, ok := reject[dstID]; ok {
				continue
			}
			newAdjacency = newAdjacency.Add(dstID)
		}

		// Good.
		node.Adjacency = newAdjacency
		output[id] = node
	}
	return output
}

func isSystem(md map[string]string) bool {
	containerName := md[docker.ContainerName]
	if _, ok := systemContainerNames[containerName]; ok {
		return true
	}
	imagePrefix := strings.SplitN(md[docker.ImageName], ":", 2)[0] // :(
	if _, ok := systemImagePrefixes[imagePrefix]; ok {
		return true
	}
	if md[docker.LabelPrefix+"works.weave.role"] == "system" {
		return true
	}
	return false
}

var systemContainerNames = map[string]struct{}{
	"weavescope": {},
	"weavedns":   {},
	"weave":      {},
	"weaveproxy": {},
	"weaveexec":  {},
	"ecs-agent":  {},
}

var systemImagePrefixes = map[string]struct{}{
	"weaveworks/scope":        {},
	"weaveworks/weavedns":     {},
	"weaveworks/weave":        {},
	"weaveworks/weaveproxy":   {},
	"weaveworks/weaveexec":    {},
	"amazon/amazon-ecs-agent": {},
}
