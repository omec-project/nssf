// Copyright 2024 Canonical Ltd.
//
// SPDX-License-Identifier: Apache-2.0
package consumer

import (
	"testing"

	"github.com/omec-project/nssf/context"
	"github.com/omec-project/openapi/v2/models"
)

const testNfId = "test-id"

func TestBuildNFProfile_EmptyContext(t *testing.T) {
	ctx := context.NSSFContext{NfId: testNfId}

	profile, err := getNfProfile(&ctx, []models.PlmnId{})
	if err != nil {
		t.Errorf("Error building NFProfile: %v", err)
	}

	if profile.GetNfInstanceId() != testNfId ||
		profile.GetNfType() != models.NFTYPE_NSSF ||
		profile.GetNfStatus() != models.NFSTATUS_REGISTERED ||
		len(profile.GetPlmnList()) != 0 ||
		profile.GetIpv4Addresses()[0] != ctx.RegisterIPv4 ||
		profile.GetNfServices() != nil {
		t.Errorf("Unexpected NfProfile built: %v", profile)
	}
}

func TestBuildNFProfile_InitializedContext(t *testing.T) {
	ctx := context.NSSFContext{
		NfId:         testNfId,
		RegisterIPv4: "127.0.0.42",
		NfService: map[models.ServiceName]models.NFService{models.SERVICENAME_NNSSF_NSSELECTION: {
			ServiceInstanceId: "instance-id",
			ServiceName:       "service-name",
		}},
	}

	profile, err := getNfProfile(&ctx, []models.PlmnId{{Mcc: "200", Mnc: "99"}})
	if err != nil {
		t.Errorf("Error building NFProfile: %v", err)
	}

	if profile.GetNfInstanceId() != testNfId ||
		profile.GetNfType() != models.NFTYPE_NSSF ||
		profile.GetNfStatus() != models.NFSTATUS_REGISTERED ||
		profile.GetPlmnList()[0].GetMcc() != "200" ||
		profile.GetPlmnList()[0].GetMnc() != "99" ||
		profile.GetIpv4Addresses()[0] != ctx.RegisterIPv4 ||
		profile.GetNfServices()[0].GetServiceName() != "service-name" {
		t.Errorf("Unexpected NfProfile built: %v", profile)
	}
}
