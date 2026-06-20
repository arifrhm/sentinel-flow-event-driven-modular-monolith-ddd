package ingest

import (
	"errors"
	"net"
	"strings"
)

// ErrConsentRequired is returned when an event requires GDPR consent but doesn't have it.
var ErrConsentRequired = errors.New("GDPR consent is required for tracking personal behavioral events")

// AnonymizeIP masks the last octet of an IPv4 address or the last group of an IPv6 address.
func AnonymizeIP(ipStr string) string {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return "0.0.0.0"
	}

	ipv4 := ip.To4()
	if ipv4 != nil {
		ipv4[3] = 0
		return ipv4.String()
	}

	ipv6 := ip.To16()
	for i := 8; i < 16; i++ {
		ipv6[i] = 0
	}
	return ipv6.String()
}

// CheckGDPRConsent verifies if behavioral tracking consent has been granted.
func CheckGDPRConsent(eventType string, gdprConsent bool) error {
	if eventType == "click" || eventType == "view_item" {
		if !gdprConsent {
			return ErrConsentRequired
		}
	}
	return nil
}
