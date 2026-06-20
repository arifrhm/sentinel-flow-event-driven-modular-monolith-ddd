package marketing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"sentinel-flow/pkg/resilience"
)

// HTTPCRMAdapter implements CRMAdapter using standard HTTP JSON payload integration.
type HTTPCRMAdapter struct {
	crmURL    string
	crmClient *resilience.ResilientClient
}

// NewHTTPCRMAdapter instantiates a new HTTPCRMAdapter.
func NewHTTPCRMAdapter(crmURL string, client *resilience.ResilientClient) *HTTPCRMAdapter {
	return &HTTPCRMAdapter{
		crmURL:    crmURL,
		crmClient: client,
	}
}

// SyncContact serializes the CRMContact data and sends a POST request resiliently.
func (h *HTTPCRMAdapter) SyncContact(ctx context.Context, contact *CRMContact) error {
	payload, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.crmURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.crmClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
