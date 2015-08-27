package kubernetes

import (
	"strings"
	"time"

	"github.com/weaveworks/scope/report"
	"k8s.io/kubernetes/pkg/api"
)

const (
	PodID           = "kubernetes_pod_id"
	PodName         = "kubernetes_pod_name"
	PodCreated      = "kubernetes_pod_created"
	PodContainerIDs = "kubernetes_pod_container_ids"
)

// Pod represents a Kubernetes pod
type Pod interface {
	ID() string
	Name() string
	Namespace() string
	ContainerIDs() []string
	GetNode() report.Node
}

type pod struct {
	*api.Pod
	Node *api.Node
}

func NewPod(p *api.Pod) Pod {
	return &pod{Pod: p}
}

func (p *pod) ID() string {
	return p.ObjectMeta.Namespace + "/" + p.ObjectMeta.Name
}

func (p *pod) Name() string {
	return p.ObjectMeta.Name
}

func (p *pod) Namespace() string {
	return p.ObjectMeta.Namespace
}

func (p *pod) Created() string {
	return p.ObjectMeta.CreationTimestamp.Format(time.RFC822)
}

func (p *pod) ContainerIDs() []string {
	ids := []string{}
	for _, container := range p.Status.ContainerStatuses {
		ids = append(ids, strings.TrimPrefix(container.ContainerID, "docker://"))
	}
	return ids
}

func (p *pod) GetNode() report.Node {
	return report.MakeNodeWith(map[string]string{
		PodID:           p.ID(),
		PodName:         p.Name(),
		Namespace:       p.Namespace(),
		PodCreated:      p.Created(),
		PodContainerIDs: strings.Join(p.ContainerIDs(), ", "),
	})
}
