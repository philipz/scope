package kubernetes

import (
	"sync"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type client struct {
	client *unversioned.Client

	lock      sync.Mutex
	listeners map[chan<- watch.Event]listener
}

// TODO: proper config, maybe nil works?
// TODO: Config should probably be from unversioned.InClusterConfig(), but that
// needs to run inside a pod.
func NewClient(api string) (Client, error) {
	c, err := unversioned.New(&unversioned.Config{Host: api})
	if err != nil {
		return nil, err
	}
	return &client{
		client:    c,
		listeners: map[chan<- watch.Event]listener{},
	}, nil
}

// TODO: Subscribe to services too
func (k *client) AddEventListener(c chan<- watch.Event) error {
	pods, err := k.client.Pods(api.NamespaceAll).Watch(labels.Everything(), fields.Everything(), "")
	if err != nil {
		return err
	}
	services, err := k.client.Services(api.NamespaceAll).Watch(labels.Everything(), fields.Everything(), "")
	if err != nil {
		pods.Stop()
		return err
	}
	nodes, err := k.client.Nodes().Watch(labels.Everything(), fields.Everything(), "")
	if err != nil {
		pods.Stop()
		services.Stop()
		return err
	}
	k.lock.Lock()
	defer k.lock.Unlock()
	k.removeEventListener(c)
	l := listener{pods: pods, services: services, nodes: nodes}
	k.listeners[c] = l
	go l.Listen(c)

	return nil
}

func (k *client) RemoveEventListener(c chan watch.Event) error {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.removeEventListener(c)
}

func (k *client) removeEventListener(c chan<- watch.Event) error {
	l, ok := k.listeners[c]
	if ok {
		delete(k.listeners, c)
		l.Stop()
	}
	return nil
}

type listener struct {
	pods, services, nodes watch.Interface
}

func (l listener) Listen(c chan<- watch.Event) {
	podsChan := l.pods.ResultChan()
	servicesChan := l.services.ResultChan()
	nodesChan := l.nodes.ResultChan()
	for {
		select {
		case event, ok := <-podsChan:
			if !ok {
				return
			}
			c <- event
		case event, ok := <-servicesChan:
			if !ok {
				return
			}
			c <- event
		case event, ok := <-nodesChan:
			if !ok {
				return
			}
			c <- event
		}
	}
}

func (l listener) Stop() {
	l.pods.Stop()
	l.services.Stop()
	l.nodes.Stop()
}
