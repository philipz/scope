package docker_test

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	client "github.com/fsouza/go-dockerclient"

	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/test"
)

type mockConnection struct {
	reader *io.PipeReader
}

func (c *mockConnection) Do(req *http.Request) (resp *http.Response, err error) {
	return &http.Response{
		Body: c.reader,
	}, nil
}

func (c *mockConnection) Close() error {
	return c.reader.Close()
}

func TestContainer(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	oldDialStub, oldNewClientConnStub := docker.DialStub, docker.NewClientConnStub
	defer func() { docker.DialStub, docker.NewClientConnStub = oldDialStub, oldNewClientConnStub }()

	docker.DialStub = func(network, address string) (net.Conn, error) {
		return nil, nil
	}

	reader, writer := io.Pipe()
	connection := &mockConnection{reader}

	docker.NewClientConnStub = func(c net.Conn, r *bufio.Reader) docker.ClientConn {
		return connection
	}

	c := docker.NewContainer(container1)
	err := c.StartGatheringStats()
	if err != nil {
		t.Errorf("%v", err)
	}
	defer c.StopGatheringStats()

	// Send some stats to the docker container
	stats := &client.Stats{}
	stats.MemoryStats.Usage = 12345
	if err = json.NewEncoder(writer).Encode(&stats); err != nil {
		t.Error(err)
	}

	// Now see if we go them
	test.Poll(t, 100*time.Millisecond, "12345", func() interface{} {
		return c.GetNode().Metadata[docker.MemoryUsage]
	})
}
