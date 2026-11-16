// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2024 Canonical Ltd.
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"testing"

	"github.com/omec-project/util/nfconfig"
)

func TestCheckConfigVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "matching version", version: NSSF_EXPECTED_CONFIG_VERSION, wantErr: false},
		{name: "mismatched version", version: "2.0.0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := NssfConfig
			defer func() { NssfConfig = orig }()

			NssfConfig = Config{Info: &Info{Version: tt.version}}
			err := CheckConfigVersion()
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckConfigVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWebuiUrl(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		want       string
	}{
		{
			name:       "default webui URL",
			configFile: "../test/conf/test_nssf_config.yaml",
			want:       nfconfig.DefaultWebuiUri,
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
