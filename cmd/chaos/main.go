package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nikshrma/heimdall/internal/config"
)

// EventLog represents a single structured chaos event line in JSONL format.
type EventLog struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"` // "kill", "revive", "skip"
	Target    string `json:"target"`
	Mode      string `json:"mode"`
	Reason    string `json:"reason,omitempty"`
}

// TargetSummary contains downtime metrics for a specific target container.
type TargetSummary struct {
	Name              string  `json:"name"`
	KillCount         int     `json:"kill_count"`
	TotalDowntimeSecs float64 `json:"total_downtime_seconds"`
}

// SummaryLog represents the final run summary object appended to the JSONL log file.
type SummaryLog struct {
	Event     string          `json:"event"`
	Timestamp string          `json:"timestamp"`
	Targets   []TargetSummary `json:"targets"`
}

type chaosRunner struct {
	cli               *client.Client
	targets           []string
	seed              int64
	minInterval       float64
	maxInterval       float64
	duration          float64
	mode              string
	minAlive          int
	logFile           string
	grafanaURL        string
	rng               *rand.Rand
	logWriter         *os.File
	stoppedContainers map[string]bool
	killCounts        map[string]int
	downtimeStart     map[string]time.Time
	totalDowntime     map[string]time.Duration
}

func extractContainerName(backend string) string {
	u, err := url.Parse(backend)
	if err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	host := backend
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func (r *chaosRunner) logEvent(action, target, reason string) {
	nowStr := time.Now().UTC().Format(time.RFC3339)
	evt := EventLog{
		Timestamp: nowStr,
		Action:    action,
		Target:    target,
		Mode:      r.mode,
		Reason:    reason,
	}

	if r.logWriter != nil {
		data, err := json.Marshal(evt)
		if err == nil {
			r.logWriter.Write(append(data, '\n'))
		}
	}
}

func (r *chaosRunner) annotateGrafana(text string, tags []string) {
	if r.grafanaURL == "" {
		return
	}
	payload := map[string]interface{}{
		"text": text,
		"tags": tags,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := strings.TrimRight(r.grafanaURL, "/") + "/api/annotations"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func (r *chaosRunner) revive(ctx context.Context, target string) {
	var err error
	var eventText string
	if r.mode == "pause" {
		err = r.cli.ContainerUnpause(ctx, target)
		eventText = fmt.Sprintf("REVIVED (unpaused) container: %s", target)
	} else {
		err = r.cli.ContainerStart(ctx, target, container.StartOptions{})
		eventText = fmt.Sprintf("REVIVED (started) container: %s", target)
	}

	if err != nil {
		fmt.Printf("[Chaos Error] Failed reviving %s: %v\n", target, err)
		return
	}

	if startTime, ok := r.downtimeStart[target]; ok {
		r.totalDowntime[target] += time.Since(startTime)
		delete(r.downtimeStart, target)
	}
	delete(r.stoppedContainers, target)

	fmt.Printf("[Chaos Event] %s\n", eventText)
	r.logEvent("revive", target, "")
	r.annotateGrafana(eventText, []string{"chaos", "revive", target})
}

func (r *chaosRunner) kill(ctx context.Context, target string) {
	var err error
	var eventText string
	if r.mode == "pause" {
		err = r.cli.ContainerPause(ctx, target)
		eventText = fmt.Sprintf("KILLED (paused) container: %s", target)
	} else {
		timeout := 1
		err = r.cli.ContainerStop(ctx, target, container.StopOptions{Timeout: &timeout})
		eventText = fmt.Sprintf("KILLED (stopped) container: %s", target)
	}

	if err != nil {
		fmt.Printf("[Chaos Error] Failed killing %s: %v\n", target, err)
		return
	}

	r.stoppedContainers[target] = true
	r.killCounts[target]++
	r.downtimeStart[target] = time.Now()

	fmt.Printf("[Chaos Event] %s\n", eventText)
	r.logEvent("kill", target, "")
	r.annotateGrafana(eventText, []string{"chaos", "kill", target})
}

func (r *chaosRunner) cleanupAndExit() {
	fmt.Println("\n[Chaos] Cleaning up target containers...")
	cleanupTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for target := range r.stoppedContainers {
		if startTime, ok := r.downtimeStart[target]; ok {
			r.totalDowntime[target] += cleanupTime.Sub(startTime)
			delete(r.downtimeStart, target)
		}
		if r.mode == "pause" {
			_ = r.cli.ContainerUnpause(ctx, target)
		} else {
			_ = r.cli.ContainerStart(ctx, target, container.StartOptions{})
		}
		fmt.Printf("[Chaos] Restored %s\n", target)
	}

	fmt.Println("\n================ CHAOS RUN SUMMARY ================")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TARGET\tKILLS\tTOTAL DOWNTIME")

	sortedTargets := make([]string, len(r.targets))
	copy(sortedTargets, r.targets)
	sort.Strings(sortedTargets)

	var summaries []TargetSummary
	for _, target := range sortedTargets {
		dt := r.totalDowntime[target]
		fmt.Fprintf(w, "%s\t%d\t%s\n", target, r.killCounts[target], dt.Round(time.Millisecond))
		summaries = append(summaries, TargetSummary{
			Name:              target,
			KillCount:         r.killCounts[target],
			TotalDowntimeSecs: dt.Seconds(),
		})
	}
	w.Flush()
	fmt.Println("===================================================")

	if r.logWriter != nil {
		summaryLog := SummaryLog{
			Event:     "summary",
			Timestamp: cleanupTime.UTC().Format(time.RFC3339),
			Targets:   summaries,
		}
		data, err := json.Marshal(summaryLog)
		if err == nil {
			r.logWriter.Write(append(data, '\n'))
		}
		r.logWriter.Close()
	}

	os.Exit(0)
}

func main() {
	var (
		targetsFlag     string
		configFlag      string
		seedFlag        int64
		minIntervalFlag float64
		maxIntervalFlag float64
		durationFlag    float64
		modeFlag        string
		minAliveFlag    int
		forceFlag       bool
		logFileFlag     string
		grafanaFlag     string
	)

	flag.StringVar(&targetsFlag, "targets", "im1,im2,im3", "Comma-separated list of target Docker container names")
	flag.StringVar(&configFlag, "config", "", "Path to Heimdall YAML route config file to extract backend targets from")
	flag.Int64Var(&seedFlag, "seed", 42, "RNG seed for deterministic reproducible chaos runs")
	flag.Float64Var(&minIntervalFlag, "min-interval", 3.0, "Minimum sleep interval between chaos events in seconds")
	flag.Float64Var(&maxIntervalFlag, "max-interval", 8.0, "Maximum sleep interval between chaos events in seconds")
	flag.Float64Var(&durationFlag, "duration", 60.0, "Total duration of the chaos run in seconds (0 for indefinite)")
	flag.StringVar(&modeFlag, "mode", "stop", "Chaos mode: 'stop' (stop/start) or 'pause' (pause/unpause)")
	flag.IntVar(&minAliveFlag, "min-alive", 1, "Minimum number of targets that must remain active")
	flag.BoolVar(&forceFlag, "force", false, "Force run even if min-alive >= number of targets")
	flag.StringVar(&logFileFlag, "log-file", "chaos-events.jsonl", "Path to output structured JSONL event log")
	flag.StringVar(&grafanaFlag, "grafana", "http://localhost:2999", "Grafana URL for pushing event annotations (empty to disable)")

	flag.Parse()

	if minIntervalFlag < 0 || maxIntervalFlag < minIntervalFlag {
		fmt.Printf("Invalid interval range: min-interval (%.2f) must be >= 0 and <= max-interval (%.2f)\n", minIntervalFlag, maxIntervalFlag)
		os.Exit(1)
	}

	var targets []string
	if configFlag != "" {
		cfg, err := config.Load(configFlag)
		if err != nil {
			fmt.Printf("Failed to load config file %s: %v\n", configFlag, err)
			os.Exit(1)
		}
		targetMap := make(map[string]bool)
		for _, route := range cfg.Routes {
			for _, backend := range route.Backends {
				name := extractContainerName(backend)
				if name != "" {
					targetMap[name] = true
				}
			}
		}
		for t := range targetMap {
			targets = append(targets, t)
		}
		sort.Strings(targets)
	} else {
		for _, t := range strings.Split(targetsFlag, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				targets = append(targets, trimmed)
			}
		}
	}

	if len(targets) == 0 {
		fmt.Println("Error: No target containers specified.")
		os.Exit(1)
	}

	if minAliveFlag >= len(targets) {
		fmt.Printf("Warning: --min-alive (%d) is >= total number of targets (%d).\n"+
			"Under this configuration, every kill operation will be skipped and the run will only idle or revive containers.\n",
			minAliveFlag, len(targets))
		if !forceFlag {
			fmt.Println("Aborting run. Use --force to override and run anyway.")
			os.Exit(1)
		}
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Failed to create Docker client: %v\n", err)
		os.Exit(1)
	}

	var logWriter *os.File
	if logFileFlag != "" {
		dir := filepath.Dir(logFileFlag)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		f, err := os.OpenFile(logFileFlag, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file %s: %v\n", logFileFlag, err)
			os.Exit(1)
		}
		logWriter = f
	}

	runner := &chaosRunner{
		cli:               cli,
		targets:           targets,
		seed:              seedFlag,
		minInterval:       minIntervalFlag,
		maxInterval:       maxIntervalFlag,
		duration:          durationFlag,
		mode:              modeFlag,
		minAlive:          minAliveFlag,
		logFile:           logFileFlag,
		grafanaURL:        grafanaFlag,
		rng:               rand.New(rand.NewSource(seedFlag)),
		logWriter:         logWriter,
		stoppedContainers: make(map[string]bool),
		killCounts:        make(map[string]int),
		downtimeStart:     make(map[string]time.Time),
		totalDowntime:     make(map[string]time.Duration),
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})

	go func() {
		<-sigChan
		close(done)
	}()

	fmt.Printf("[Chaos] Starting chaos run (Targets: %v, Seed: %d, Mode: %s, MinAlive: %d)\n",
		runner.targets, runner.seed, runner.mode, runner.minAlive)
	fmt.Printf("[Chaos] Random interval range: %.2fs - %.2fs, Duration: %.2fs\n",
		runner.minInterval, runner.maxInterval, runner.duration)

	ctx := context.Background()
	startTime := time.Now()

	for {
		select {
		case <-done:
			runner.cleanupAndExit()
			return
		default:
		}

		if runner.duration > 0 && time.Since(startTime).Seconds() >= runner.duration {
			fmt.Println("[Chaos] Duration reached.")
			runner.cleanupAndExit()
			return
		}

		target := runner.targets[runner.rng.Intn(len(runner.targets))]
		if runner.stoppedContainers[target] {
			runner.revive(ctx, target)
		} else {
			activeCount := len(runner.targets) - len(runner.stoppedContainers)
			if activeCount-1 < runner.minAlive {
				reason := fmt.Sprintf("min-alive safety floor reached (%d active, min %d)", activeCount, runner.minAlive)
				fmt.Printf("[Chaos Event] SKIPPED killing container %s (reason: %s)\n", target, reason)
				runner.logEvent("skip", target, reason)

				if len(runner.stoppedContainers) > 0 {
					var downList []string
					for d := range runner.stoppedContainers {
						downList = append(downList, d)
					}
					sort.Strings(downList)
					reviveTarget := downList[runner.rng.Intn(len(downList))]
					runner.revive(ctx, reviveTarget)
				}
			} else {
				runner.kill(ctx, target)
			}
		}

		sleepSec := runner.minInterval
		if runner.maxInterval > runner.minInterval {
			sleepSec += runner.rng.Float64() * (runner.maxInterval - runner.minInterval)
		}
		sleepDuration := time.Duration(sleepSec * float64(time.Second))

		if runner.duration > 0 {
			remaining := time.Duration(runner.duration*float64(time.Second)) - time.Since(startTime)
			if remaining <= 0 {
				fmt.Println("[Chaos] Duration reached.")
				runner.cleanupAndExit()
				return
			}
			if sleepDuration > remaining {
				sleepDuration = remaining
			}
		}

		select {
		case <-done:
			runner.cleanupAndExit()
			return
		case <-time.After(sleepDuration):
		}
	}
}
