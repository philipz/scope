package kubernetes

import "github.com/weaveworks/scope/report"

// Reporter generate Reports containing Container and ContainerImage topologies
type Reporter struct {
	registry Registry
}

// NewReporter makes a new Reporter
func NewReporter(registry Registry) *Reporter {
	return &Reporter{
		registry: registry,
	}
}

// Report generates a Report containing Container and ContainerImage topologies
func (r *Reporter) Report() (report.Report, error) {
	result := report.MakeReport()
	result.Pod = result.Pod.Merge(r.podTopology())
	result.Service = result.Service.Merge(r.serviceTopology())
	return result, nil
}

func (r *Reporter) podTopology() report.Topology {
	result := report.MakeTopology()
	r.registry.WalkPods(func(p Pod) {
		nodeID := report.MakePodNodeID(p.Namespace(), p.ID())
		result.Nodes[nodeID] = p.GetNode()
	})
	return result
}

func (r *Reporter) serviceTopology() report.Topology {
	result := report.MakeTopology()
	r.registry.WalkServices(func(s Service) {
		nodeID := report.MakeServiceNodeID(s.Namespace(), s.ID())
		result.Nodes[nodeID] = s.GetNode()
	})
	return result
}
