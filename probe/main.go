package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/weaveworks/procspy"
	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/probe/endpoint"
	"github.com/weaveworks/scope/probe/host"
	"github.com/weaveworks/scope/probe/overlay"
	"github.com/weaveworks/scope/probe/process"
	"github.com/weaveworks/scope/probe/sniff"
	"github.com/weaveworks/scope/report"
	"github.com/weaveworks/scope/xfer"
)

var version = "dev" // set at build time

func main() {
	var (
		httpListen         = flag.String("http.listen", "", "listen address for HTTP profiling and instrumentation server")
		publishInterval    = flag.Duration("publish.interval", 3*time.Second, "publish (output) interval")
		spyInterval        = flag.Duration("spy.interval", time.Second, "spy (scan) interval")
		listen             = flag.String("listen", ":"+strconv.Itoa(xfer.ProbePort), "listen address")
		prometheusEndpoint = flag.String("prometheus.endpoint", "/metrics", "Prometheus metrics exposition endpoint (requires -http.listen)")
		spyProcs           = flag.Bool("processes", true, "report processes (needs root)")
		dockerEnabled      = flag.Bool("docker", false, "collect Docker-related attributes for processes")
		dockerInterval     = flag.Duration("docker.interval", 10*time.Second, "how often to update Docker attributes")
		dockerBridge       = flag.String("docker.bridge", "docker0", "the docker bridge name")
		weaveRouterAddr    = flag.String("weave.router.addr", "", "IP address or FQDN of the Weave router")
		procRoot           = flag.String("proc.root", "/proc", "location of the proc filesystem")
		captureEnabled     = flag.Bool("capture", false, "perform sampled packet capture")
		captureInterfaces  = flag.String("capture.interfaces", interfaces(), "packet capture on these interfaces")
		captureOn          = flag.Duration("capture.on", 1*time.Second, "packet capture duty cycle 'on'")
		captureOff         = flag.Duration("capture.window", 5*time.Second, "packet capture duty cycle 'off'")
	)
	flag.Parse()
	log.SetFlags(log.Lmicroseconds)

	if len(flag.Args()) != 0 {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("probe version %s", version)

	procspy.SetProcRoot(*procRoot)

	if *httpListen != "" {
		log.Printf("profiling data being exported to %s", *httpListen)
		log.Printf("go tool pprof http://%s/debug/pprof/{profile,heap,block}", *httpListen)
		if *prometheusEndpoint != "" {
			log.Printf("exposing Prometheus endpoint at %s%s", *httpListen, *prometheusEndpoint)
			http.Handle(*prometheusEndpoint, makePrometheusHandler())
		}
		go func() {
			err := http.ListenAndServe(*httpListen, nil)
			log.Print(err)
		}()
	}

	if *spyProcs && os.Getegid() != 0 {
		log.Printf("warning: process reporting enabled, but that requires root to find everything")
	}

	publisher, err := xfer.NewTCPPublisher(*listen)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	var (
		hostName     = hostname()
		hostID       = hostName // TODO: we should sanitize the hostname
		taggers      = []Tagger{newTopologyTagger(), host.NewTagger(hostID)}
		reporters    = []Reporter{host.NewReporter(hostID, hostName), endpoint.NewReporter(hostID, hostName, *spyProcs)}
		processCache *process.CachingWalker
	)

	processCache = process.NewCachingWalker(process.NewWalker(*procRoot))
	reporters = append(reporters, process.NewReporter(processCache, hostID))

	if *dockerEnabled {
		if err := report.AddLocalBridge(*dockerBridge); err != nil {
			log.Fatalf("failed to get docker bridge address: %v", err)
		}

		dockerRegistry, err := docker.NewRegistry(*dockerInterval)
		if err != nil {
			log.Fatalf("failed to start docker registry: %v", err)
		}
		defer dockerRegistry.Stop()

		taggers = append(taggers, docker.NewTagger(dockerRegistry, processCache))
		reporters = append(reporters, docker.NewReporter(dockerRegistry, hostID))
	}

	if *weaveRouterAddr != "" {
		weave, err := overlay.NewWeave(*weaveRouterAddr)
		if err != nil {
			log.Fatalf("failed to start Weave tagger: %v", err)
		}
		taggers = append(taggers, weave)
		reporters = append(reporters, weave)
	}

	if *captureEnabled {
		if *captureOn > *captureOff {
			log.Fatalf("-capture.on (%s) must be <= -capture.off (%s)", *captureOn, *captureOff)
		}
		for _, iface := range strings.Split(*captureInterfaces, ",") {
			source, err := sniff.NewSource(iface)
			if err != nil {
				log.Printf("warning: %v", err)
				continue
			}
			log.Printf("capturing packets on %s", iface)
			reporters = append(reporters, sniff.New(hostID, source, *captureOn, *captureOff))
		}
	}

	log.Printf("listening on %s", *listen)

	quit := make(chan struct{})
	defer close(quit)
	go func() {
		var (
			pubTick = time.Tick(*publishInterval)
			spyTick = time.Tick(*spyInterval)
			r       = report.MakeReport()
		)

		for {
			select {
			case <-pubTick:
				publishTicks.WithLabelValues().Add(1)
				publisher.Publish(r)
				r = report.MakeReport()

			case <-spyTick:
				if err := processCache.Update(); err != nil {
					log.Printf("error reading processes: %v", err)
				}

				for _, reporter := range reporters {
					newReport, err := reporter.Report()
					if err != nil {
						log.Printf("error generating report: %v", err)
					}
					r.Merge(newReport)
				}

				r = Apply(r, taggers)

			case <-quit:
				return
			}
		}
	}()

	log.Printf("%s", <-interrupt())
}

func interrupt() chan os.Signal {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}

func interfaces() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Print(err)
		return ""
	}
	a := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		a = append(a, iface.Name)
	}
	return strings.Join(a, ",")
}
