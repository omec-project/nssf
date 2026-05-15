// SPDX-FileCopyrightText: 2025 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
//

package util

import (
	"testing"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

func TestCheckSupportedSnssaiInPlmn(t *testing.T) {
	plmn1 := models.PlmnId{Mcc: "001", Mnc: "01"}
	plmn2 := models.PlmnId{Mcc: "002", Mnc: "02"}

	snssai1 := models.Snssai{Sst: 4, Sd: openapi.PtrString("000001")}
	snssai2 := models.Snssai{Sst: 2, Sd: openapi.PtrString("000002")}
	snssai3 := models.Snssai{Sst: 4}
	standardSnssai := models.Snssai{Sst: 2}

	supportedNssai := factory.SupportedNssaiInPlmn{
		plmn1: {factory.SnssaiToKey(snssai1): struct{}{}},
	}

	tests := []struct {
		name     string
		snssai   models.Snssai
		plmnId   models.PlmnId
		expected bool
	}{
		{
			name:     "Supported S-NSSAI in PLMN",
			snssai:   snssai1,
			plmnId:   plmn1,
			expected: true,
		},
		{
			name:     "Unsupported S-NSSAI in PLMN",
			snssai:   snssai2,
			plmnId:   plmn1,
			expected: false,
		},
		{
			name:     "PLMN ID not configured",
			snssai:   snssai1,
			plmnId:   plmn2,
			expected: false,
		},
		{
			name:     "Equal SST but different SD",
			snssai:   snssai3,
			plmnId:   plmn1,
			expected: false,
		},
		{
			name:     "Standard S-NSSAI is valid",
			snssai:   standardSnssai,
			plmnId:   plmn1,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFactoryConfig := factory.NssfConfig
			defer func() {
				factory.NssfConfig = originalFactoryConfig
			}()
			factory.NssfConfig = factory.Config{
				Configuration: &factory.Configuration{
					SupportedNssaiInPlmnList: supportedNssai,
				},
			}

			result := CheckSupportedSnssaiInPlmn(tc.snssai, tc.plmnId)
			if result != tc.expected {
				t.Errorf("Expected CheckSupportedSnssaiInPlmn to be `%v`, got `%v`", tc.expected, result)
			}
		})
	}
}

func TestCheckSupportedSnssaiInPlmn_EmptyConfig(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()
	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: factory.SupportedNssaiInPlmn{},
		},
	}

	plmn := models.PlmnId{Mcc: "001", Mnc: "01"}
	snssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("000001")}

	result := CheckSupportedSnssaiInPlmn(snssai, plmn)
	if result != false {
		t.Errorf("Expected CheckSupportedSnssaiInPlmn to be false, got `%v`", result)
	}
}

func TestCheckSupportedSnssaiInPlmn_EmptySupportedNssaiButExistingPlmn(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()

	plmn := models.PlmnId{Mcc: "001", Mnc: "01"}
	snssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("000001")}

	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: factory.SupportedNssaiInPlmn{plmn: map[factory.SnssaiKey]struct{}{}},
		},
	}

	result := CheckSupportedSnssaiInPlmn(snssai, plmn)
	if result != false {
		t.Errorf("Expected CheckSupportedSnssaiInPlmn to be false, got `%v`", result)
	}
}

func TestCheckSupportedSnssaiInTa_UsesValueEqualityForSd(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()

	tai := models.Tai{
		PlmnId: models.PlmnId{Mcc: "001", Mnc: "01"},
		Tac:    "000001",
	}
	configuredSnssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}
	requestSnssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}

	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			TaList: []factory.TaConfig{
				{
					Tai:                 &tai,
					SupportedSnssaiList: []models.Snssai{configuredSnssai},
				},
			},
		},
	}

	if !CheckSupportedSnssaiInTa(requestSnssai, tai) {
		t.Fatal("expected S-NSSAI with equal SST/SD values to be supported in TA")
	}
}

func TestCheckSupportedNssaiInPlmn(t *testing.T) {
	plmn := models.PlmnId{Mcc: "001", Mnc: "01"}

	snssai1 := models.Snssai{Sst: 1, Sd: openapi.PtrString("000001")}
	snssai2 := models.Snssai{Sst: 2, Sd: openapi.PtrString("000002")}
	snssai3 := models.Snssai{Sst: 3, Sd: openapi.PtrString("000003")}

	supportedNssai := factory.SupportedNssaiInPlmn{
		plmn: {factory.SnssaiToKey(snssai1): struct{}{}, factory.SnssaiToKey(snssai2): struct{}{}},
	}

	tests := []struct {
		name     string
		nssai    []models.Snssai
		plmnId   models.PlmnId
		expected bool
	}{
		{
			name:     "All S-NSSAIs supported",
			nssai:    []models.Snssai{snssai1, snssai2},
			plmnId:   plmn,
			expected: true,
		},
		{
			name:     "Subset of supported S-NSSAIs",
			nssai:    []models.Snssai{snssai2},
			plmnId:   plmn,
			expected: true,
		},
		{
			name:     "One unsupported S-NSSAI",
			nssai:    []models.Snssai{snssai1, snssai3},
			plmnId:   plmn,
			expected: false,
		},
		{
			name:     "PLMN ID not in config",
			nssai:    []models.Snssai{snssai1},
			plmnId:   models.PlmnId{Mcc: "999", Mnc: "99"},
			expected: false,
		},
		{
			name:     "Empty NSSAI list",
			nssai:    []models.Snssai{},
			plmnId:   plmn,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalFactoryConfig := factory.NssfConfig
			defer func() {
				factory.NssfConfig = originalFactoryConfig
			}()
			factory.NssfConfig = factory.Config{
				Configuration: &factory.Configuration{
					SupportedNssaiInPlmnList: supportedNssai,
				},
			}

			result := CheckSupportedNssaiInPlmn(tc.nssai, tc.plmnId)
			if result != tc.expected {
				t.Errorf("Expected CheckSupportedNssaiInPlmn to be `%v`, got `%v`", tc.expected, result)
			}
		})
	}
}

func TestCheckSupportedNssaiInPlmn_EmptyConfig(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()
	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: factory.SupportedNssaiInPlmn{},
		},
	}

	plmn := models.PlmnId{Mcc: "001", Mnc: "01"}
	snssai := []models.Snssai{{Sst: 1, Sd: openapi.PtrString("000001")}}

	result := CheckSupportedNssaiInPlmn(snssai, plmn)
	if result != false {
		t.Errorf("Expected CheckSupportedNssaiInPlmn to be false, got `%v`", result)
	}
}

func TestCheckSupportedNssaiInPlmn_EmptySupportedNssaiButExistingPlmn(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()

	plmn := models.PlmnId{Mcc: "001", Mnc: "01"}
	snssai := []models.Snssai{{Sst: 1, Sd: openapi.PtrString("000001")}}

	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			SupportedNssaiInPlmnList: factory.SupportedNssaiInPlmn{plmn: map[factory.SnssaiKey]struct{}{}},
		},
	}

	result := CheckSupportedNssaiInPlmn(snssai, plmn)
	if result != false {
		t.Errorf("Expected CheckSupportedNssaiInPlmn to be false, got `%v`", result)
	}
}

func TestGetNsiInformationListFromConfig_UsesValueEqualityForSd(t *testing.T) {
	originalFactoryConfig := factory.NssfConfig
	defer func() {
		factory.NssfConfig = originalFactoryConfig
	}()

	configuredSnssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}
	requestSnssai := models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")}
	expected := []models.NsiInformation{{NrfId: "nrf-1"}}

	factory.NssfConfig = factory.Config{
		Configuration: &factory.Configuration{
			NsiList: []factory.NsiConfig{{
				Snssai:             &configuredSnssai,
				NsiInformationList: expected,
			}},
		},
	}

	result := GetNsiInformationListFromConfig(requestSnssai)
	if len(result) != len(expected) {
		t.Fatalf("expected %d NSI info entries, got %d", len(expected), len(result))
	}
	if result[0].GetNrfId() != expected[0].GetNrfId() {
		t.Fatalf("expected NRF ID %q, got %q", expected[0].GetNrfId(), result[0].GetNrfId())
	}
}
