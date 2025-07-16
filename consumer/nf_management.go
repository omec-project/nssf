// SPDX-FileCopyrightText: 2025 Canonical Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Consumer
 *
 * Network Function Management
 */

package consumer

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	nssfContext "github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi"
	"github.com/omec-project/openapi/Nnrf_NFManagement"
	"github.com/omec-project/openapi/models"
)

func getNfProfile(currentNssfContext *nssfContext.NSSFContext, plmnConfig []models.PlmnId) (profile models.NfProfile, err error) {
	if currentNssfContext == nil {
		return models.NfProfile{}, fmt.Errorf("nssf context has not been intialized. NF profile cannot be built")
	}
	profile.NfInstanceId = currentNssfContext.NfId
	profile.NfType = models.NfType_NSSF
	profile.NfStatus = models.NfStatus_REGISTERED
	if len(plmnConfig) > 0 {
		plmnCopy := make([]models.PlmnId, len(plmnConfig))
		copy(plmnCopy, plmnConfig)
		profile.PlmnList = &plmnCopy
	}
	profile.Ipv4Addresses = []string{currentNssfContext.RegisterIPv4}
	var services []models.NfService
	for _, nfService := range currentNssfContext.NfService {
		services = append(services, nfService)
	}
	if len(services) > 0 {
		profile.NfServices = &services
	}
	return profile, err
}

var SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (prof models.NfProfile, resourceNrfUri string, err error) {
	nssfSelf := nssfContext.NSSF_Self()
	nfProfile, err := getNfProfile(nssfSelf, plmnConfig)
	if err != nil {
		return models.NfProfile{}, "", err
	}
	configuration := Nnrf_NFManagement.NewConfiguration()
	configuration.SetBasePath(nssfSelf.NrfUri)
	apiClient := Nnrf_NFManagement.NewAPIClient(configuration)

	receivedNfProfile, res, err := apiClient.NFInstanceIDDocumentApi.RegisterNFInstance(context.TODO(), nfProfile.NfInstanceId, nfProfile)
	logger.ConsumerLog.Debugf("RegisterNFInstance done using profile: %+v", nfProfile)

	if err != nil {
		return models.NfProfile{}, "", err
	}
	if res == nil {
		return models.NfProfile{}, "", fmt.Errorf("no response from server")
	}

	switch res.StatusCode {
	case http.StatusOK: // NFUpdate
		logger.ConsumerLog.Debugln("NSSF NF profile updated with complete replacement")
		return receivedNfProfile, "", nil
	case http.StatusCreated: // NFRegister
		resourceUri := res.Header.Get("Location")
		resourceNrfUri = resourceUri[:strings.Index(resourceUri, "/nnrf-nfm/")]
		retrieveNfInstanceId := resourceUri[strings.LastIndex(resourceUri, "/")+1:]
		nssfSelf.NfId = retrieveNfInstanceId
		logger.ConsumerLog.Debugln("NSSF NF profile registered to the NRF")
		return receivedNfProfile, resourceNrfUri, nil
	default:
		return receivedNfProfile, "", fmt.Errorf("unexpected status code returned by the NRF %d", res.StatusCode)
	}
}

var SendUpdateNFInstance = func(patchItem []models.PatchItem) (models.NfProfile, *models.ProblemDetails, error) {
	logger.ConsumerLog.Debugln("send Update NFInstance")

	nssfSelf := nssfContext.NSSF_Self()
	configuration := Nnrf_NFManagement.NewConfiguration()
	configuration.SetBasePath(nssfSelf.NrfUri)
	client := Nnrf_NFManagement.NewAPIClient(configuration)

	var res *http.Response
	receivedNfProfile, res, err := client.NFInstanceIDDocumentApi.UpdateNFInstance(context.Background(), nssfSelf.NfId, patchItem)
	if err != nil {
		if openapiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if model := openapiErr.Model(); model != nil {
				if problem, ok := model.(models.ProblemDetails); ok {
					return models.NfProfile{}, &problem, nil
				}
			}
		}
		return models.NfProfile{}, nil, err
	}

	if res == nil {
		return models.NfProfile{}, nil, fmt.Errorf("no response from server")
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		return receivedNfProfile, nil, nil
	}
	return models.NfProfile{}, nil, fmt.Errorf("unexpected response code")
}

var SendDeregisterNFInstance = func() error {
	logger.AppLog.Infoln("send Deregister NFInstance")

	nssfSelf := nssfContext.NSSF_Self()
	// Set client and set url
	configuration := Nnrf_NFManagement.NewConfiguration()
	configuration.SetBasePath(nssfSelf.NrfUri)
	client := Nnrf_NFManagement.NewAPIClient(configuration)

	res, err := client.NFInstanceIDDocumentApi.DeregisterNFInstance(context.Background(), nssfSelf.NfId)
	if err != nil {
		return err
	}
	if res == nil {
		return fmt.Errorf("no response from server")
	}
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("unexpected response code")
}
