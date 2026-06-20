package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"sentinel-flow/pkg/broker"
)

var (
	ingestURL    = "http://localhost:8081/v1/events"
	marketingURL = "http://localhost:8082/v1"
	analyticsURL = "http://localhost:8083/v1"
)

func main() {
	log.Println("===================================================")
	log.Println("           SENTINEL-FLOW TRAFFIC SIMULATOR         ")
	log.Println("===================================================")

	client := &http.Client{Timeout: 2 * time.Second}

	// 1. Wait a moment to ensure cluster is online
	log.Println("[Simulator] Checking if cluster services are active...")
	if err := pingCluster(client); err != nil {
		log.Fatalf("[Simulator] Cluster not ready. Ensure orchestrator is running: %v", err)
	}

	// 2. Run test suite
	runLegitimateTraffic(client)
	runBotTraffic(client)
	runRateLimitTraffic(client)
	runGeoVelocityAnomaly(client)
	runGDPRAndCCPAFlows(client)

	// Wait 2 seconds for event queues and CRM calls to finalize
	log.Println("[Simulator] Waiting 2s for background services to complete updates...")
	time.Sleep(2 * time.Second)

	// 3. Print Final telemetry
	printDashboard(client)
}

func pingCluster(client *http.Client) error {
	var err error
	for i := 0; i < 5; i++ {
		resp, pingErr := client.Get(analyticsURL + "/dashboard")
		if pingErr == nil {
			resp.Body.Close()
			return nil
		}
		err = pingErr
		time.Sleep(500 * time.Millisecond)
	}
	return err
}

func postEvent(client *http.Client, event *broker.TrackingEvent) int {
	body, _ := json.Marshal(event)
	req, _ := http.NewRequest(http.MethodPost, ingestURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return 500
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func printDashboard(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 6: Querying Analytics Real-Time Dashboard...")
	resp, err := client.Get(analyticsURL + "/dashboard")
	if err != nil {
		log.Printf("[Simulator] Error fetching dashboard: %v", err)
		return
	}
	defer resp.Body.Close()

	dashboard, _ := io.ReadAll(resp.Body)
	fmt.Println(string(dashboard))
}
