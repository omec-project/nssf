// Copyright (c) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

func TestParseQueryParameterSupportsExplodedRegistrationRequest(t *testing.T) {
	expectedSliceInfo := models.NewSliceInfoForRegistration()
	subscribedSnssai := models.NewSubscribedSnssai(models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")})
	subscribedSnssai.SetDefaultIndication(true)
	expectedSliceInfo.SetSubscribedNssai([]models.SubscribedSnssai{*subscribedSnssai})
	expectedSliceInfo.SetRequestedNssai([]models.Snssai{{Sst: 1, Sd: openapi.PtrString("112233")}})
	mappingOfSnssai := models.NewMappingOfSnssai(models.Snssai{Sst: 1, Sd: openapi.PtrString("112233")}, models.Snssai{Sst: 2, Sd: openapi.PtrString("445566")})
	expectedSliceInfo.SetMappingOfNssai([]models.MappingOfSnssai{*mappingOfSnssai})
	expectedSliceInfo.SetRequestMapping(true)

	expectedHomePlmnID := models.NewPlmnId("001", "01")
	expectedTai := models.NewTai(*expectedHomePlmnID, "000001")
	expectedTai.SetNid("123456789AB")

	query := url.Values{}
	openapi.ParameterAddToHeaderOrQuery(query, "slice-info-request-for-registration", expectedSliceInfo, "", "")
	openapi.ParameterAddToHeaderOrQuery(query, "home-plmn-id", expectedHomePlmnID, "", "")
	openapi.ParameterAddToHeaderOrQuery(query, "tai", expectedTai, "", "")

	if len(query["slice-info-request-for-registration[requestedNssai][sst]"]) != 1 {
		t.Fatalf("expected exploded requestedNssai query keys, got %v", query)
	}
	if len(query["home-plmn-id[mcc]"]) != 1 {
		t.Fatalf("expected exploded home-plmn-id query keys, got %v", query)
	}
	if len(query["tai[plmnId][mcc]"]) != 1 {
		t.Fatalf("expected exploded tai query keys, got %v", query)
	}
	if len(query["tai[nid]"]) != 1 {
		t.Fatalf("expected exploded tai nid query key, got %v", query)
	}

	param, err := parseQueryParameter(query)
	if err != nil {
		t.Fatalf("parseQueryParameter returned error: %v", err)
	}
	if !reflect.DeepEqual(param.SliceInfoRequestForRegistration, expectedSliceInfo) {
		t.Fatalf("unexpected slice info: %+v", param.SliceInfoRequestForRegistration)
	}
	if !reflect.DeepEqual(param.HomePlmnId, expectedHomePlmnID) {
		t.Fatalf("unexpected home plmn id: %+v", param.HomePlmnId)
	}
	if !reflect.DeepEqual(param.Tai, expectedTai) {
		t.Fatalf("unexpected tai: %+v", param.Tai)
	}
}

func TestParseExplodedMappingOfSnssaiListPreservesIndexes(t *testing.T) {
	query := url.Values{
		"mappingOfNssai[servingSnssai][sst]": {"", "1"},
		"mappingOfNssai[homeSnssai][sst]":    {"2", "3"},
	}

	mappingOfNssai, err := parseExplodedMappingOfSnssaiList(query, "mappingOfNssai")
	if err != nil {
		t.Fatalf("parseExplodedMappingOfSnssaiList returned error: %v", err)
	}

	expected := []models.MappingOfSnssai{
		{
			HomeSnssai: models.Snssai{Sst: 2},
		},
		{
			ServingSnssai: models.Snssai{Sst: 1},
			HomeSnssai:    models.Snssai{Sst: 3},
		},
	}

	if !reflect.DeepEqual(mappingOfNssai, expected) {
		t.Fatalf("unexpected mappingOfNssai: %+v", mappingOfNssai)
	}
}

func TestParseExplodedSnssaiListRejectsSdWithoutSst(t *testing.T) {
	query := url.Values{
		"requestedNssai[sd]": {"010203"},
	}

	_, err := parseExplodedSnssaiList(query, "requestedNssai")
	if err == nil {
		t.Fatal("expected error for sd without sst")
	}
	if !strings.Contains(err.Error(), "requestedNssai") {
		t.Fatalf("expected error to include parameter name, got %q", err)
	}
}

func TestParseExplodedSnssaiRejectsSdWithoutSst(t *testing.T) {
	query := url.Values{
		"sNssai[sd]": {"010203"},
	}

	_, found, err := parseExplodedSnssai(query, "sNssai")
	if err == nil {
		t.Fatal("expected error for sd without sst")
	}
	if found {
		t.Fatal("expected snssai not to be marked found on invalid input")
	}
	if !strings.Contains(err.Error(), "sNssai") {
		t.Fatalf("expected error to include parameter name, got %q", err)
	}
}
