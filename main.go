package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ProbeResult holds the outcome of a single health check against one target URL.
type ProbeResult struct {
	URL        string        `json:"url"`
	Up         bool          `json:"up"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency_ms"`
	CheckedAt  time.Time     `json:"checked_at"`
	Error      string        `json:"error,omitempty"`
}

// Prober runs concurrent HTTP health checks against a set of target URLs.
type Prober struct {
	targets []string
	timeout time.Duration
	mu      sync.RWMutex
	results map[string]*ProbeResult
}

func NewProber(targets []string, timeout time.Duration) *Prober {
	return &Prober{
		targets: targets,
		timeout: timeout,
		results: make(map[string]*ProbeResult),
	}
}

// probe checks a single URL and stores the result.
func (p *Prober) probe(url string) {
	client := &http.Client{Timeout: p.timeout}
	start := time.Now()

	result := &ProbeResult{
		URL:       url,
		CheckedAt: start,
	}

	resp, err := client.Get(url)
	result.Latency = time.Since(start)

	if err != nil {
		result.Up = false
		result.Error = err.Error()
	} else {
		defer resp.Body.Close()
		result.StatusCode = resp.StatusCode
		result.Up = resp.StatusCode >= 200 && resp.StatusCode < 400
	}

	p.mu.Lock()
	p.results[url] = result
	p.mu.Unlock()
}

// RunOnce fires all probes concurrently and waits for all to finish.
func (p *Prober) RunOnce() {
	var wg sync.WaitGroup
	for _, target := range p.targets {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			p.probe(url)
		}(target)
	}
	wg.Wait()
}

// RunLoop probes continuously on the given interval until the process exits.
func (p *Prober) RunLoop(interval time.Duration) {
	for {
		p.RunOnce()
		log.Printf("probe cycle complete, next in %s", interval)
		time.Sleep(interval)
	}
}

// Results returns a snapshot of all current probe results.
func (p *Prober) Results() []*ProbeResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*ProbeResult, 0, len(p.results))
	for _, r := range p.results {
		out = append(out, r)
	}
	return out
}

// handleStatus serves the current probe results as JSON.
func (p *Prober) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	results := p.Results()

	allUp := true
	for _, r := range results {
		if !r.Up {
			allUp = false
			break
		}
	}
	if !allUp {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"all_up":  allUp,
		"targets": results,
	})
}

// handleMetrics serves Prometheus-format metrics for all probed targets.
//
// Exposes two metric families:
//
//	probe_up{url}         — 1 if the target responded with 2xx/3xx, 0 otherwise
//	probe_latency_ms{url} — round-trip latency in milliseconds
func (p *Prober) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	results := p.Results()
	var sb strings.Builder

	sb.WriteString("# HELP probe_up Whether the HTTP probe succeeded (1 = up, 0 = down)\n")
	sb.WriteString("# TYPE probe_up gauge\n")
	for _, res := range results {
		up := 0
		if res.Up {
			up = 1
		}
		fmt.Fprintf(&sb, "probe_up{url=%q} %d\n", res.URL, up)
	}

	sb.WriteString("# HELP probe_latency_ms HTTP probe round-trip latency in milliseconds\n")
	sb.WriteString("# TYPE probe_latency_ms gauge\n")
	for _, res := range results {
		if res.Up {
			fmt.Fprintf(&sb, "probe_latency_ms{url=%q} %.2f\n", res.URL, float64(res.Latency.Milliseconds()))
		}
	}

	fmt.Fprint(w, sb.String())
}

func main() {
	targetsFlag := flag.String("targets", "", "Comma-separated list of URLs to probe (required)")
	interval := flag.Duration("interval", 30*time.Second, "How often to re-probe targets")
	timeout := flag.Duration("timeout", 5*time.Second, "HTTP request timeout per probe")
	addr := flag.String("addr", ":8080", "Address to serve /metrics and /status on")
	flag.Parse()

	if *targetsFlag == "" {
		log.Fatal("--targets is required, e.g. --targets=https://example.com,https://google.com")
	}

	targets := strings.Split(*targetsFlag, ",")
	for i, t := range targets {
		targets[i] = strings.TrimSpace(t)
	}

	prober := NewProber(targets, *timeout)

	// Run the first probe cycle synchronously so /metrics isn't empty on startup.
	log.Printf("running initial probe of %d target(s)...", len(targets))
	prober.RunOnce()

	// Subsequent probes run in the background on the configured interval.
	go prober.RunLoop(*interval)

	http.HandleFunc("/metrics", prober.handleMetrics)
	http.HandleFunc("/status", prober.handleStatus)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `go-probe — endpoints: /metrics (Prometheus), /status (JSON)`)
	})

	log.Printf("go-probe listening on %s — probing every %s", *addr, *interval)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}
