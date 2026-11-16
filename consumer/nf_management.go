// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2025 Canonical Ltd
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"

	nssfContext "github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nrfclient"
)

func newNrfClient(nrfUri string) *nrfclient.Client {
	return nrfclient.New(nrfUri, nrfclient.WithLogger(logger.ConsumerLog))
}

func getNfProfile(currentNssfContext *nssfContext.NSSFContext, plmnConfig []models.PlmnId) (profile models.NFProfile, err error) {
	if currentNssfContext == nil {
		return profile, openapi.ReportError("nssf context has not been initialized. NF profile cannot be built")
	}
	profile.SetNfInstanceId(currentNssfContext.NfId)
	profile.SetNfType(models.NFTYPE_NSSF)
	profile.SetNfStatus(models.NFSTATUS_REGISTERED)
	if len(plmnConfig) > 0 {
		plmnCopy := make([]models.PlmnId, len(plmnConfig))
		copy(plmnCopy, plmnConfig)
		profile.SetPlmnList(plmnCopy)
	}
	profile.SetIpv4Addresses([]string{currentNssfContext.RegisterIPv4})
	services := map[string]models.NFService{}
	serviceList := []models.NFService{}
	for _, nfService := range currentNssfContext.NfService {
		services[nfService.GetServiceInstanceId()] = nfService
		serviceList = append(serviceList, nfService)
	}
	if len(services) > 0 {
		profile.SetNfServices(serviceList)
		profile.SetNfServiceList(services)
	}
	return profile, err
}

// SendRegisterNFInstance and SendDeregisterNFInstance take the context of the registration service, which holds
// its registration mutex while they run and cancels the context to end them when a new configuration arrives or
// the NF terminates.
var SendRegisterNFInstance = func(ctx context.Context, plmnConfig []models.PlmnId) (prof *models.NFProfile, resourceNrfUri string, err error) {
	self := nssfContext.NSSF_Self()
	nfProfile, err := getNfProfile(self, plmnConfig)
	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}
	logger.ConsumerLog.Debugf("registering NF Instance using profile: %+v", nfProfile)

	result, err := newNrfClient(self.NrfUri).Register(ctx, nfProfile)
	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}
	if result.Created {
		logger.ConsumerLog.Debugln("NSSF NF profile registered to the NRF")
		if result.NfInstanceID != "" {
			self.NfId = result.NfInstanceID
		}
	} else {
		logger.ConsumerLog.Debugln("NSSF NF profile updated with complete replacement")
	}
	return result.Profile, result.NrfURI, nil
}

var SendDeregisterNFInstance = func(ctx context.Context) error {
	logger.ConsumerLog.Infoln("send Deregister NFInstance")

	self := nssfContext.NSSF_Self()
	return newNrfClient(self.NrfUri).Deregister(ctx, self.NfId)
}

var SendUpdateNFInstance = func(ctx context.Context, patchItem []models.PatchItem) (receivedNfProfile *models.NFProfile, problemDetails *models.ProblemDetails, err error) {
	logger.ConsumerLog.Debugln("send Update NFInstance")

	self := nssfContext.NSSF_Self()
	return newNrfClient(self.NrfUri).Update(ctx, self.NfId, patchItem)
}
