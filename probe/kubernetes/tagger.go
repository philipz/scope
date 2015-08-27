package kubernetes

import (
	"github.com/weaveworks/scope/report"
)

// Tagger is a tagger that tags Kubernetes pod information to container nodes.
type Tagger struct {
	registry Registry
}

// NewTagger returns a usable Tagger.
func NewTagger(registry Registry) *Tagger {
	return &Tagger{
		registry: registry,
	}
}

// Tag implements Tagger.
// TODO: Do we even need this? We can't really correlate data between
// topologies here, as the topologies are incomplete, since they come from
// different nodes.
func (t *Tagger) Tag(r report.Report) (report.Report, error) {
	return r, nil
}
