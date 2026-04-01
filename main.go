package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
)

type ScanRequest struct {
	Target  string `json:"target"`
	AuditID string `json:"audit_id"`
}

type RawFinding struct {
	VulnerabilityName string `json:"vulnerability_name"`
	Severity          string `json:"severity"`
	MatchedAt         string `json:"matched_at"`
}

const nodeApiUrl = "http://localhost:3000/api/triage"

// Advanced CORS Middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Change "*" to your frontend URL in production
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message": "Scan initiated successfully"}`))

	// Launch the heavy scan in the background so the HTTP request doesn't block
	go runEmbeddedScan(req.Target, req.AuditID)
}

func main() {
	http.HandleFunc("/api/scan", enableCORS(scanHandler))

	fmt.Println("[GO] Origins Sentinel Engine listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runEmbeddedScan(target, auditID string) {
	fmt.Printf("[GO] Scanning %s (Audit: %s)\n", target, auditID)

	// Context with background ensures the engine doesn't timeout prematurely
	ne, err := nuclei.NewNucleiEngineCtx(context.Background(),
		nuclei.WithTemplateFilters(nuclei.TemplateFilters{
			Tags: []string{"cve", "exposure", "misconfig", "tech"},
		}),
	)
	if err != nil {
		log.Printf("[GO] Engine init failed: %v\n", err)
		return
	}
	defer ne.Close()

	ne.LoadTargets([]string{target}, false)

	// CRITICAL FIX: Initialize as an empty slice, NOT a nil slice.
	// This guarantees we send '[]' instead of 'null' in the JSON payload.
	findings := make([]RawFinding, 0)

	err = ne.ExecuteWithCallback(func(event *output.ResultEvent) {
		if event.MatcherStatus {
			findings = append(findings, RawFinding{
				VulnerabilityName: event.Info.Name,
				Severity:          event.Info.SeverityHolder.Severity.String(),
				MatchedAt:         event.Matched,
			})
			fmt.Printf("[HIT] Found %s [%s]\n", event.Info.Name, event.Info.SeverityHolder.Severity.String())
		}
	})

	if err != nil {
		log.Printf("[GO] Execution error: %v\n", err)
	}

	fmt.Printf("[GO] Scan Complete. Found %d raw hits. Forwarding to Node AI...\n", len(findings))

	payload, _ := json.Marshal(map[string]interface{}{
		"audit_id":     auditID,
		"raw_findings": findings,
	})

	resp, err := http.Post(nodeApiUrl, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[ERR] Failed to reach Node.js API: %v\n", err)
	} else {
		log.Printf("[GO] Handoff successful! Node.js returned status: %s\n", resp.Status)
		resp.Body.Close()
	}
}