// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024 Canonical Ltd.
/*
 *  Tests for NSSF Configuration Factory
 */

package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Webui URL is not set then default Webui URL value is returned
func TestGetDefaultWebuiUrl(t *testing.T) {
	if err := InitConfigFactory("../test/conf/test_nssf_config.yaml"); err != nil {
		t.Errorf("error in InitConfigFactory: %v", err)
	}
	got := NssfConfig.Configuration.WebuiUri
	want := "http://webui:5001"
	assert.Equal(t, got, want, "The webui URL is not correct.")
}

// Webui URL is set to a custom value then custom Webui URL is returned
func TestGetCustomWebuiUrl(t *testing.T) {
	if err := InitConfigFactory("../test/conf/test_nssf_config_with_custom_webui_url.yaml"); err != nil {
		t.Errorf("error in InitConfigFactory: %v", err)
	}
	got := NssfConfig.Configuration.WebuiUri
	want := "http://myspecialwebui:9872"
	assert.Equal(t, got, want, "The webui URL is not correct.")
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
