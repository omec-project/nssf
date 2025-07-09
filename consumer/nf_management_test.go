// Copyright 2024 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
package consumer

import (
	"testing"

	"github.com/omec-project/nssf/context"
	"github.com/omec-project/openapi/models"
)

func TestBuildNFProfile_EmptyContext(t *testing.T) {
	ctx := context.NSSFContext{NfId: "test-id"}

	profile, err := getNfProfile(&ctx, []models.PlmnId{})
	if err != nil {
		t.Errorf("Error building NFProfile: %v\n", err)
	}

	if profile.NfInstanceId != "test-id" ||
		profile.NfType != models.NfType_NSSF ||
		profile.NfStatus != models.NfStatus_REGISTERED ||
		profile.PlmnList != nil ||
		profile.Ipv4Addresses[0] != ctx.RegisterIPv4 ||
		profile.NfServices != nil {
		t.Errorf("Unexpected NfProfile built: %v\n", profile)
	}
}

func TestBuildNFProfile_InitializedContext(t *testing.T) {
	ctx := context.NSSFContext{
		NfId:         "test-id",
		RegisterIPv4: "127.0.0.42",
		NfService: map[models.ServiceName]models.NfService{models.ServiceName_NNSSF_NSSELECTION: {
			ServiceInstanceId: "instance-id",
			ServiceName:       "service-name",
		}},
	}

	profile, err := getNfProfile(&ctx, []models.PlmnId{{Mcc: "200", Mnc: "99"}})
	if err != nil {
		t.Errorf("Error building NFProfile: %v\n", err)
	}

	if profile.NfInstanceId != "test-id" ||
		profile.NfType != models.NfType_NSSF ||
		profile.NfStatus != models.NfStatus_REGISTERED ||
		(*profile.PlmnList)[0].Mcc != "200" ||
		(*profile.PlmnList)[0].Mnc != "99" ||
		profile.Ipv4Addresses[0] != ctx.RegisterIPv4 ||
		(*profile.NfServices)[0].ServiceName != "service-name" {
		t.Errorf("Unexpected NfProfile built: %v\n", profile)
	}
}
