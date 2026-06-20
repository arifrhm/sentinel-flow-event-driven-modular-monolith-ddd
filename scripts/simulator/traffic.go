package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"sentinel-flow/pkg/broker"
)

func runLegitimateTraffic(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 1: Sending legitimate tracking traffic...")
	
	users := []string{"usr_alpha", "usr_beta", "usr_gamma"}
	events := []string{"signup", "view_item", "cart_abandoned", "checkout_completed"}

	var wg sync.WaitGroup
	for _, u := range users {
		for _, e := range events {
			wg.Add(1)
			go func(user, event string) {
				defer wg.Done()
				consent := true
				if event == "view_item" {
					consent = true
				}
				evt := &broker.TrackingEvent{
					EventID:     fmt.Sprintf("evt-legit-%s-%d", user, time.Now().UnixNano()),
					UserID:      user,
					EventType:   event,
					IPAddress:   "192.168.1.15",
					UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
					Payload:     map[string]interface{}{"email": user + "@gmail.com", "country": "ID"},
					Timestamp:   time.Now(),
					GDPRConsent: consent,
				}
				code := postEvent(client, evt)
				if code != http.StatusAccepted {
					log.Printf("[Simulator] Legitimate event failed with status %d", code)
				}
			}(u, e)
			time.Sleep(10 * time.Millisecond)
		}
	}
	wg.Wait()
	log.Println("[Simulator] Legitimate traffic sent successfully.")
}

func runBotTraffic(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 2: Sending traffic with crawler bot User-Agents...")

	bots := []struct {
		userID string
		agent  string
	}{
		{"bot_google", "Googlebot/2.1 (+http://www.google.com/bot.html)"},
		{"bot_scraping", "HeadlessChrome/109.0.0.0 Safari/537.36"},
		{"bot_python", "python-requests/2.28.1"},
	}

	for _, b := range bots {
		evt := &broker.TrackingEvent{
			EventID:     fmt.Sprintf("evt-bot-%s-%d", b.userID, time.Now().UnixNano()),
			UserID:      b.userID,
			EventType:   "click",
			IPAddress:   "8.8.8.8",
			UserAgent:   b.agent,
			Payload:     map[string]interface{}{"target": "login_payload"},
			Timestamp:   time.Now(),
			GDPRConsent: true,
		}
		postEvent(client, evt)
	}
	log.Println("[Simulator] Bot events sent.")
}

func runRateLimitTraffic(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 3: Simulating IP Rate Limit attack (rapid clicks)...")

	ipAttack := "103.120.15.40"
	for i := 1; i <= 15; i++ {
		evt := &broker.TrackingEvent{
			EventID:     fmt.Sprintf("evt-burst-%d", time.Now().UnixNano()),
			UserID:      "usr_spammer",
			EventType:   "click",
			IPAddress:   ipAttack,
			UserAgent:   "Mozilla/5.0",
			Payload:     map[string]interface{}{"click_id": i},
			Timestamp:   time.Now(),
			GDPRConsent: true,
		}
		postEvent(client, evt)
	}
	log.Println("[Simulator] IP rate-limiting attack simulation completed.")
}

func runGeoVelocityAnomaly(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 4: Simulating Geo Velocity anomaly (impossible travel)...")

	user := "usr_traveller"
	evt1 := &broker.TrackingEvent{
		EventID:     "evt-geo-1",
		UserID:      user,
		EventType:   "click",
		IPAddress:   "114.122.0.1",
		UserAgent:   "Mozilla/5.0",
		Payload:     map[string]interface{}{"country": "ID"},
		Timestamp:   time.Now(),
		GDPRConsent: true,
	}
	postEvent(client, evt1)

	time.Sleep(100 * time.Millisecond)

	evt2 := &broker.TrackingEvent{
		EventID:     "evt-geo-2",
		UserID:      user,
		EventType:   "click",
		IPAddress:   "126.0.0.1",
		UserAgent:   "Mozilla/5.0",
		Payload:     map[string]interface{}{"country": "JP"},
		Timestamp:   time.Now(),
		GDPRConsent: true,
	}
	postEvent(client, evt2)
	log.Println("[Simulator] Geo-velocity anomaly events sent.")
}

func runGDPRAndCCPAFlows(client *http.Client) {
	log.Println("\n[Simulator] >>> STEP 5: Simulating GDPR & CCPA Compliance Operations...")

	userCompliance := "usr_privacy_advocate"

	evt := &broker.TrackingEvent{
		EventID:     "evt-comp-1",
		UserID:      userCompliance,
		EventType:   "signup",
		IPAddress:   "10.0.0.2",
		UserAgent:   "Mozilla/5.0",
		Payload:     map[string]interface{}{"email": "privacy@advocate.org"},
		Timestamp:   time.Now(),
		GDPRConsent: true,
	}
	postEvent(client, evt)
	time.Sleep(100 * time.Millisecond)

	log.Printf("[Simulator] Requesting CCPA Data Export for user '%s'...", userCompliance)
	exportURL := fmt.Sprintf("%s/privacy/ccpa-export?user_id=%s", marketingURL, userCompliance)
	resp, err := client.Get(exportURL)
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[Simulator] CCPA Export Result:\n%s", string(body))
		resp.Body.Close()
	}

	log.Printf("[Simulator] Requesting GDPR Account Deletion (Purge) for user '%s'...", userCompliance)
	deletePayload, _ := json.Marshal(map[string]string{"user_id": userCompliance})
	respDel, err := client.Post(marketingURL+"/privacy/gdpr-delete", "application/json", bytes.NewBuffer(deletePayload))
	if err == nil {
		body, _ := io.ReadAll(respDel.Body)
		log.Printf("[Simulator] GDPR Delete Result: %s", string(body))
		respDel.Body.Close()
	}

	log.Printf("[Simulator] Requesting CCPA Data Export *after* GDPR purge...")
	respPost, err := client.Get(exportURL)
	if err == nil {
		body, _ := io.ReadAll(respPost.Body)
		log.Printf("[Simulator] CCPA Export Result (Scrubbed): %s", string(body))
		respPost.Body.Close()
	}
}
