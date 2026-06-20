package ingest

import (
	"errors"
	"testing"
)

func TestAnonymizeIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.15", "192.168.1.0"},
		{"8.8.8.8", "8.8.8.0"},
		{"2001:db8:85a3:0:0:8a2e:370:7334", "2001:db8:85a3::"},
		{"invalid-ip", "0.0.0.0"},
		{"", "0.0.0.0"},
	}

	for _, tt := range tests {
		actual := AnonymizeIP(tt.input)
		if actual != tt.expected {
			t.Errorf("AnonymizeIP(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestCheckGDPRConsent(t *testing.T) {
	tests := []struct {
		eventType   string
		consent     bool
		expectedErr error
	}{
		{"click", true, nil},
		{"click", false, ErrConsentRequired},
		{"view_item", true, nil},
		{"view_item", false, ErrConsentRequired},
		{"signup", false, nil},
		{"signup", true, nil},
		{"custom_event", false, nil},
	}

	for _, tt := range tests {
		err := CheckGDPRConsent(tt.eventType, tt.consent)
		if !errors.Is(err, tt.expectedErr) {
			t.Errorf("CheckGDPRConsent(%q, %t) error = %v, expected %v", tt.eventType, tt.consent, err, tt.expectedErr)
		}
	}
}
