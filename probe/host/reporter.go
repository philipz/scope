package host

import (
	"runtime"
	"strings"
	"time"

	"github.com/weaveworks/scope/report"
)

// Keys for use in Node.Metadata.
const (
	Timestamp     = "ts"
	HostName      = "host_name"
	LocalNetworks = "local_networks"
	OS            = "os"
	Load          = "load"
	KernelVersion = "kernel_version"
	Uptime        = "uptime"
)

// Exposed for testing.
const (
	ProcUptime = "/proc/uptime"
	ProcLoad   = "/proc/loadavg"
)

// Exposed for testing.
var (
	Now = func() string { return time.Now().UTC().Format(time.RFC3339Nano) }
)

// Reporter generates Reports containing the host topology.
type Reporter struct {
	hostID    string
	hostName  string
	localNets report.Networks
}

// NewReporter returns a Reporter which produces a report containing host
// topology for this host.
func NewReporter(hostID, hostName string, localNets report.Networks) *Reporter {
	return &Reporter{
		hostID:    hostID,
		hostName:  hostName,
		localNets: localNets,
	}
}

// Report implements Reporter.
func (r *Reporter) Report() (report.Report, error) {
	var (
		rep        = report.MakeReport()
		localCIDRs []string
	)

	for _, localNet := range r.localNets {
		localCIDRs = append(localCIDRs, localNet.String())
	}

	uptime, err := GetUptime()
	if err != nil {
		return rep, err
	}

	kernel, err := GetKernelVersion()
	if err != nil {
		return rep, err
	}

	rep.Host.Nodes[report.MakeHostNodeID(r.hostID)] = report.MakeNodeWith(map[string]string{
		Timestamp:     Now(),
		HostName:      r.hostName,
		LocalNetworks: strings.Join(localCIDRs, " "),
		OS:            runtime.GOOS,
		Load:          GetLoad(),
		KernelVersion: kernel,
		Uptime:        uptime.String(),
	})

	return rep, nil
}
