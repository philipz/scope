package kubernetes

import (
	"fmt"
	"strings"
	"time"

	"github.com/weaveworks/scope/report"
	"k8s.io/kubernetes/pkg/api"
)

const (
	ServiceID      = "kubernetes_service_id"
	ServiceName    = "kubernetes_service_name"
	ServiceCreated = "kubernetes_service_created"
	ServicePorts   = "kubernetes_service_ports"
	ServiceIngress = "kubernetes_service_ingress"
)

// Service represents a Kubernetes service
type Service interface {
	ID() string
	Name() string
	Namespace() string
	GetNode() report.Node
}

type service struct {
	*api.Service
}

func NewService(s *api.Service) Service {
	return &service{Service: s}
}

func (s *service) ID() string {
	return s.ObjectMeta.Namespace + "/" + s.ObjectMeta.Name
}

func (s *service) Name() string {
	return s.ObjectMeta.Name
}

func (s *service) Namespace() string {
	return s.ObjectMeta.Namespace
}

func (s *service) GetNode() report.Node {
	return report.MakeNodeWith(map[string]string{
		ServiceID:      s.ID(),
		ServiceName:    s.Name(),
		ServiceCreated: s.ObjectMeta.CreationTimestamp.Format(time.RFC822),
		Namespace:      s.Namespace(),
		ServicePorts:   s.ports(),
		ServiceIngress: s.ingress(),
	})
}

func (s *service) ports() string {
	if ports := s.Spec.Ports; len(ports) > 0 {
		forwards := []string{}
		for _, port := range ports {
			forwards = append(forwards, fmt.Sprintf("%d/%s -> %s", port.Port, port.Protocol, port.TargetPort.String()))
		}
		return strings.Join(forwards, ", ")
	}
	return ""
}

func (s *service) ingress() string {
	if ingress := s.Status.LoadBalancer.Ingress; len(ingress) > 0 {
		ips := []string{}
		for _, ip := range ingress {
			if ip.IP != "" {
				ips = append(ips, ip.IP)
			} else {
				ips = append(ips, ip.Hostname)
			}
		}
		return strings.Join(ips, ", ")
	}
	return ""
}
