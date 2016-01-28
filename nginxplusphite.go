package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/quipo/statsd"
)

// NginxStats is a Json parsing struct to match nginx status
type NginxStats struct {
	Version       int    `json:"version"`
	NginxVersion  string `json:"nginx_version"`
	Address       string `json:"address"`
	Generation    int    `json:"generation"`
	LoadTimestamp int64  `json:"load_timestamp"`
	Timestamp     int64  `json:"timestamp"`
	Pid           int    `json:"pid"`
	Processes     struct {
		Respawned int `json:"respawned"`
	} `json:"processes"`
	Connections struct {
		Accepted int `json:"accepted"`
		Dropped  int `json:"dropped"`
		Active   int `json:"active"`
		Idle     int `json:"idle"`
	} `json:"connections"`
	Ssl struct {
		Handshakes       int `json:"handshakes"`
		HandshakesFailed int `json:"handshakes_failed"`
		SessionReuses    int `json:"session_reuses"`
	} `json:"ssl"`
	Requests struct {
		Total   int `json:"total"`
		Current int `json:"current"`
	} `json:"requests"`
	ServerZones struct {
	} `json:"server_zones"`
	Upstreams struct {
		CacheServers struct {
			Peers []struct {
				ID        int    `json:"id"`
				Server    string `json:"server"`
				Backup    bool   `json:"backup"`
				Weight    int    `json:"weight"`
				State     string `json:"state"`
				Active    int    `json:"active"`
				Requests  int    `json:"requests"`
				Responses struct {
					OneXx   int `json:"1xx"`
					TwoXx   int `json:"2xx"`
					ThreeXx int `json:"3xx"`
					FourXx  int `json:"4xx"`
					FiveXx  int `json:"5xx"`
					Total   int `json:"total"`
				} `json:"responses"`
				Sent         int   `json:"sent"`
				Received     int64 `json:"received"`
				Fails        int   `json:"fails"`
				Unavail      int   `json:"unavail"`
				HealthChecks struct {
					Checks     int  `json:"checks"`
					Fails      int  `json:"fails"`
					Unhealthy  int  `json:"unhealthy"`
					LastPassed bool `json:"last_passed"`
				} `json:"health_checks"`
				Downtime  int   `json:"downtime"`
				Downstart int   `json:"downstart"`
				Selected  int64 `json:"selected"`
			} `json:"peers"`
			Keepalive int `json:"keepalive"`
		} `json:"cache_servers"`
	} `json:"upstreams"`
	Caches struct {
	} `json:"caches"`
}

var (
	host       string
	port       int
	metricPath string
	interval   int
	url        string
	version    int
)

func init() {
	flag.StringVar(&host, "H", "localhost", "Hostname for statsd")
	flag.IntVar(&port, "p", 8125, "Port for statsd")
	flag.StringVar(&metricPath, "m", "nginx.stats", "Metric path")
	flag.IntVar(&interval, "i", 10, "Check stats each <i> seconds")
	flag.StringVar(&url, "u", "http://localhost/status", "Nginx plus status URL")
	flag.IntVar(&version, "v", 5, "NGinx JSON version")
}

func main() {

	tenSecs := time.Duration(interval) * time.Second

	if len(os.Args) < 4 {
		flag.Usage()
		os.Exit(127)
	}
	flag.Parse()

	c := statsd.NewStatsdClient(fmt.Sprintf("%s:%d", host, port), metricPath)

	for {
		log.Printf("Running...")
		work(c)
		log.Printf("Done! Sleeping for %d seconds", interval)
		time.Sleep(tenSecs)
	}

}

func work(c *statsd.StatsdClient) {
	err := c.CreateSocket()
	if err != nil {
		log.Fatal("Error creatig socket")
	}

	defer c.Close()

	// Read results from status URL
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		log.Fatalf("%v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("%v", err)
	}

	var data NginxStats
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Send stats to statsd
	_ = c.Gauge("connections.accepted", int64(data.Connections.Accepted))
	_ = c.Gauge("connections.active", int64(data.Connections.Active))
	_ = c.Gauge("connections.dropped", int64(data.Connections.Dropped))
	_ = c.Gauge("connections.idle", int64(data.Connections.Idle))
	_ = c.Gauge("requests.total", int64(data.Requests.Total))
	_ = c.Gauge("requests.current", int64(data.Requests.Current))

	for v := range data.Upstreams.CacheServers.Peers {
		sPath := fmt.Sprintf("upstreams.cache_servers.%d.", v)
		_ = c.Gauge(sPath+"active", int64(data.Upstreams.CacheServers.Peers[v].Active))
		_ = c.Gauge(sPath+"requests", int64(data.Upstreams.CacheServers.Peers[v].Requests))
		_ = c.Gauge(sPath+"fails", int64(data.Upstreams.CacheServers.Peers[v].Fails))
		_ = c.Gauge(sPath+"unavail", int64(data.Upstreams.CacheServers.Peers[v].Unavail))
		_ = c.Gauge(sPath+"sent", int64(data.Upstreams.CacheServers.Peers[v].Sent))
		_ = c.Gauge(sPath+"received", int64(data.Upstreams.CacheServers.Peers[v].Sent))

		_ = c.Gauge(sPath+"responses.1xx", int64(data.Upstreams.CacheServers.Peers[v].Responses.OneXx))
		_ = c.Gauge(sPath+"responses.2xx", int64(data.Upstreams.CacheServers.Peers[v].Responses.TwoXx))
		_ = c.Gauge(sPath+"responses.3xx", int64(data.Upstreams.CacheServers.Peers[v].Responses.ThreeXx))
		_ = c.Gauge(sPath+"responses.4xx", int64(data.Upstreams.CacheServers.Peers[v].Responses.FourXx))
		_ = c.Gauge(sPath+"responses.5xx", int64(data.Upstreams.CacheServers.Peers[v].Responses.FiveXx))
		_ = c.Gauge(sPath+"responses.total", int64(data.Upstreams.CacheServers.Peers[v].Responses.Total))

		_ = c.Gauge(sPath+"health_checks.fails", int64(data.Upstreams.CacheServers.Peers[v].HealthChecks.Fails))
		_ = c.Gauge(sPath+"health_checks.unhealthy", int64(data.Upstreams.CacheServers.Peers[v].HealthChecks.Unhealthy))
	}
}
