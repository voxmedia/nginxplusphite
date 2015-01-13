package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/quipo/statsd"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type NginxStats struct {
	Address     string   `json:"address"`
	Caches      struct{} `json:"caches"`
	Connections struct {
		Accepted float64 `json:"accepted"`
		Active   float64 `json:"active"`
		Dropped  float64 `json:"dropped"`
		Idle     float64 `json:"idle"`
	} `json:"connections"`
	LoadTimestamp float64 `json:"load_timestamp"`
	NginxVersion  string  `json:"nginx_version"`
	Requests      struct {
		Current float64 `json:"current"`
		Total   float64 `json:"total"`
	} `json:"requests"`
	ServerZones struct{} `json:"server_zones"`
	Timestamp   float64  `json:"timestamp"`
	Upstreams   struct {
		CacheServers []struct {
			Active       float64 `json:"active"`
			Backup       bool    `json:"backup"`
			Downstart    float64 `json:"downstart"`
			Downtime     float64 `json:"downtime"`
			Fails        float64 `json:"fails"`
			HealthChecks struct {
				Checks     float64 `json:"checks"`
				Fails      float64 `json:"fails"`
				LastPassed bool    `json:"last_passed"`
				Unhealthy  float64 `json:"unhealthy"`
			} `json:"health_checks"`
			ID        float64 `json:"id"`
			Keepalive float64 `json:"keepalive"`
			Received  float64 `json:"received"`
			Requests  float64 `json:"requests"`
			Responses struct {
				OneH   float64 `json:"1xx"`
				TwoH   float64 `json:"2xx"`
				ThreeH float64 `json:"3xx"`
				FourH  float64 `json:"4xx"`
				FiveH  float64 `json:"5xx"`
				Total  float64 `json:"total"`
			} `json:"responses"`
			Selected float64 `json:"selected"`
			Sent     float64 `json:"sent"`
			Server   string  `json:"server"`
			State    string  `json:"state"`
			Unavail  float64 `json:"unavail"`
			Weight   float64 `json:"weight"`
		} `json:"cache_servers"`
	} `json:"upstreams"`
	Version float64 `json:"version"`
}

var (
	host       string
	port       int
	metricPath string
	interval   int
	url        string
)

func init() {
	flag.StringVar(&host, "H", "localhost", "Hostname for statsd")
	flag.IntVar(&port, "p", 8125, "Port for statsd")
	flag.StringVar(&metricPath, "m", "nginx.stats", "Metric path")
	flag.IntVar(&interval, "i", 10, "Check stats each <i> seconds")
	flag.StringVar(&url, "u", "http://localhost/status", "Nginx plus status URL")
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
	c.CreateSocket()
	defer c.Close()

	// Read results from status URL
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var data NginxStats
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Send stats to statsd
	c.Total("connections.accepted", int64(data.Connections.Accepted))
	c.Gauge("connections.active", int64(data.Connections.Active))
	c.Total("connections.dropped", int64(data.Connections.Dropped))
	c.Gauge("connections.idle", int64(data.Connections.Idle))
	c.Total("requests.total", int64(data.Requests.Total))
	c.Gauge("requests.current", int64(data.Requests.Current))

	for v := range data.Upstreams.CacheServers {
		sPath := fmt.Sprintf("upstreams.cache_servers.%d.", v)
		c.Gauge(sPath+"active", int64(data.Upstreams.CacheServers[v].Active))
		c.Total(sPath+"requests", int64(data.Upstreams.CacheServers[v].Requests))
		c.Total(sPath+"fails", int64(data.Upstreams.CacheServers[v].Fails))
		c.Total(sPath+"unavail", int64(data.Upstreams.CacheServers[v].Unavail))
		c.Gauge(sPath+"sent", int64(data.Upstreams.CacheServers[v].Sent))
		c.Gauge(sPath+"received", int64(data.Upstreams.CacheServers[v].Sent))

		c.Total(sPath+"responses.1xx", int64(data.Upstreams.CacheServers[v].Responses.OneH))
		c.Total(sPath+"responses.2xx", int64(data.Upstreams.CacheServers[v].Responses.TwoH))
		c.Total(sPath+"responses.3xx", int64(data.Upstreams.CacheServers[v].Responses.ThreeH))
		c.Total(sPath+"responses.4xx", int64(data.Upstreams.CacheServers[v].Responses.FourH))
		c.Total(sPath+"responses.5xx", int64(data.Upstreams.CacheServers[v].Responses.FiveH))
		c.Total(sPath+"responses.total", int64(data.Upstreams.CacheServers[v].Responses.Total))

		c.Total(sPath+"health_checks.fails", int64(data.Upstreams.CacheServers[v].HealthChecks.Fails))
		c.Total(sPath+"health_checks.unhealthy", int64(data.Upstreams.CacheServers[v].HealthChecks.Unhealthy))
	}
}
