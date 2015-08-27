package kubernetes

import (
	"log"
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"
)

const (
	Namespace = "kubernetes_namespace"
)

// Vars exported for testing.
var (
	NewClientStub  = NewClient
	NewPodStub     = NewPod
	NewServiceStub = NewService
)

// Registry keeps track of running kubernetes pods and services
type Registry interface {
	Stop()
	WalkPods(f func(Pod))
	WalkServices(f func(Service))
}

type registry struct {
	sync.RWMutex
	quit     chan chan struct{}
	interval time.Duration
	client   Client

	pods     map[string]Pod
	services map[string]Service
	nodes    map[string]*api.Node
}

// Client interface for mocking.
type Client interface {
	AddEventListener(chan<- watch.Event) error
	RemoveEventListener(chan watch.Event) error
}

// NewRegistry returns a usable Registry. Don't forget to Stop it.
func NewRegistry(apiAddr string, interval time.Duration) (Registry, error) {
	client, err := NewClientStub(apiAddr)
	if err != nil {
		return nil, err
	}

	r := &registry{
		pods:     map[string]Pod{},
		services: map[string]Service{},
		nodes:    map[string]*api.Node{},

		client:   client,
		interval: interval,
		quit:     make(chan chan struct{}),
	}

	go r.loop()
	return r, nil
}

// Stop stops the Docker registry's event subscriber.
func (r *registry) Stop() {
	ch := make(chan struct{})
	r.quit <- ch
	<-ch
}

func (r *registry) loop() {
	for {
		// NB listenForEvents blocks.
		// Returning false means we should exit.
		if !r.listenForEvents() {
			return
		}

		// Sleep here so we don't hammer the
		// logs if docker is down
		time.Sleep(r.interval)
	}
}

func (r *registry) listenForEvents() bool {
	// First we empty the store lists.
	// This ensure anything that went away inbetween calls to
	// listenForEvents don't hang around.
	r.reset()

	// Next, start listening for events.  We do this before fetching
	// the list of stuff so we don't miss containers created
	// after listing but before listening for events.
	events := make(chan watch.Event)
	if err := r.client.AddEventListener(events); err != nil {
		log.Printf("kubernetes registry: %s", err)
		return true
	}
	defer func() {
		if err := r.client.RemoveEventListener(events); err != nil {
			log.Printf("kubernetes registry: %s", err)
		}
	}()

	for {
		select {
		case event := <-events:
			r.handleEvent(event)

		case ch := <-r.quit:
			r.Lock()
			defer r.Unlock()

			close(ch)
			return false
		}
	}
}

func (r *registry) reset() {
	r.Lock()
	defer r.Unlock()

	r.pods = map[string]Pod{}
	r.services = map[string]Service{}
	r.nodes = map[string]*api.Node{}
}

func (r *registry) handleEvent(event watch.Event) {
	r.Lock()
	defer r.Unlock()
	switch object := event.Object.(type) {
	case *api.Pod:
		id := object.ObjectMeta.Name
		switch event.Type {
		case watch.Deleted:
			delete(r.pods, id)
			log.Printf("kubernetes pod: stopped collecting stats for %s", id)
		default:
			if _, ok := r.pods[id]; !ok {
				log.Printf("kubernetes pod: collecting stats for %s", id)
			}
			r.pods[id] = NewPodStub(object)
		}
	case *api.Service:
		id := object.ObjectMeta.Name
		switch event.Type {
		case watch.Deleted:
			delete(r.services, id)
			log.Printf("kubernetes service: stopped collecting stats for %s", id)
		default:
			if _, ok := r.services[id]; !ok {
				log.Printf("kubernetes service: collecting stats for %s", id)
			}
			r.services[id] = NewServiceStub(object)
		}
	case *api.Node:
		id := object.ObjectMeta.Name
		switch event.Type {
		case watch.Deleted:
			delete(r.nodes, id)
			log.Printf("kubernetes node: stopped collecting stats for %s", id)
		default:
			if _, ok := r.nodes[id]; !ok {
				log.Printf("kubernetes node: collecting stats for %s", id)
			}
			r.nodes[id] = object
		}
	default:
		log.Printf("Unknown kubernetes event: %v", event)
	}
}

func (r *registry) WalkPods(f func(Pod)) {
	r.RLock()
	defer r.RUnlock()
	for _, pod := range r.pods {
		f(pod)
	}
}

func (r *registry) WalkServices(f func(Service)) {
	r.RLock()
	defer r.RUnlock()
	for _, service := range r.services {
		f(service)
	}
}
