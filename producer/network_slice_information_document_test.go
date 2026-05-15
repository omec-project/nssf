// Copyright (c) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

func TestParseQueryParameterSupportsExplodedRegistrationRequest(t *testing.T) {
	expectedSliceInfo := models.NewSliceInfoForRegistration()
	expectedSliceInfo.SubscribedNssai = []models.SubscribedSnssai{{
		SubscribedSnssai:  models.Snssai{Sst: 1, Sd: openapi.PtrString("010203")},
		DefaultIndication: openapi.PtrBool(true),
	}}
	expectedSliceInfo.RequestedNssai = []models.Snssai{{Sst: 1, Sd: openapi.PtrString("112233")}}
	expectedSliceInfo.MappingOfNssai = []models.MappingOfSnssai{{
		ServingSnssai: models.Snssai{Sst: 1, Sd: openapi.PtrString("112233")},
		HomeSnssai:    models.Snssai{Sst: 2, Sd: openapi.PtrString("445566")},
	}}
	expectedSliceInfo.RequestMapping = openapi.PtrBool(true)

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
