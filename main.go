package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

func main() {
	// Define command-line flags
	botURL := flag.String("bot-url", "", "Telegram Bot API URL")
	pushURL := flag.String("push-url", "http://127.0.0.1:9091/", "Pushgateway URL")
	interval := flag.Duration("interval", time.Minute*1, "Interval between checks")

	flag.Parse()

	if *botURL == "" {
		log.Fatal("Bot URL must be provided")
	}
	jobName := "oula"

	log.Printf("Starting Telegram Bot monitoring with Bot URL: %s, Pushgateway URL: %s, Interval: %s, Job Name: %s", *botURL, *pushURL, *interval, jobName)

	// Define Prometheus Gauge
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tg_bot_status",
		Help: "Status of the Telegram bot (1 for OK, 0 for error)",
	})

	for {
		log.Println("Checking bot status...")
		status, err := checkBotStatus(*botURL)
		if err != nil {
			log.Printf("Error checking bot status: %v", err)
			gauge.Set(0)
		} else {
			if status.Ok {
				log.Println("Bot status: OK")
				gauge.Set(1)
			} else {
				log.Printf("Bot error: %s", status.Description)
				gauge.Set(0)
			}
		}

		pushMetrics(*pushURL, jobName, gauge)

		log.Printf("Sleeping for %s before the next check", *interval)
		time.Sleep(*interval)
	}
}

type BotStatus struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
}

func checkBotStatus(url string) (*BotStatus, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var status BotStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &status, nil
}

func pushMetrics(pushURL, jobName string, gauge prometheus.Gauge) {
	log.Printf("Pushing metrics to Pushgateway at %s with job name %s...", pushURL, jobName)
	if err := push.New(pushURL, jobName).
		Collector(gauge).
		Push(); err != nil {
		log.Printf("Could not push to Pushgateway: %v", err)
	} else {
		log.Println("Pushed metrics successfully")
	}
}
