// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd.
/*
 *  Tests for NSSF Configuration Factory
 */

package factory

import (
	"testing"
)

func TestWebuiUrl(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		want       string
	}{
		{
			name:       "default webui URL",
			configFile: "../test/conf/test_nssf_config.yaml",
			want:       "http://webui:5001",
		},
		{
			name:       "custom webui URL",
			configFile: "../test/conf/test_nssf_config_with_custom_webui_url.yaml",
			want:       "http://myspecialwebui:9872",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original config
			origNssfConfig := NssfConfig
			defer func() { NssfConfig = origNssfConfig }()

			if err := InitConfigFactory(tt.configFile); err != nil {
				t.Fatalf("error in InitConfigFactory: %v", err)
			}

			got := NssfConfig.Configuration.WebuiUri
			if got != tt.want {
				t.Errorf("The webui URL is not correct. got = %q, want = %q", got, tt.want)
			}
		})
	}
}

func TestValidateWebuiUri(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		isValid bool
	}{
		{
			name:    "valid https URI with port",
			uri:     "https://webui:5001",
			isValid: true,
		},
		{
			name:    "valid http URI with port",
			uri:     "http://webui:5001",
			isValid: true,
		},
		{
			name:    "valid https URI without port",
			uri:     "https://webui",
			isValid: true,
		},
		{
			name:    "valid http URI without port",
			uri:     "http://webui.com",
			isValid: true,
		},
		{
			name:    "invalid host",
			uri:     "http://:8080",
			isValid: false,
		},
		{
			name:    "invalid scheme",
			uri:     "ftp://webui:21",
			isValid: false,
		},
		{
			name:    "missing scheme",
			uri:     "webui:9090",
			isValid: false,
		},
		{
			name:    "missing host",
			uri:     "https://",
			isValid: false,
		},
		{
			name:    "empty string",
			uri:     "",
			isValid: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebuiUri(tc.uri)
			if err == nil && !tc.isValid {
				t.Errorf("expected URI: %s to be invalid", tc.uri)
			}
			if err != nil && tc.isValid {
				t.Errorf("expected URI: %s to be valid", tc.uri)
			}
		})
	}
}
