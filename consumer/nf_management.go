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
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/Nnrf_NFManagement"
	"github.com/omec-project/openapi/v2/models"
)

func getNfProfile(currentNssfContext *nssfContext.NSSFContext, plmnConfig []models.PlmnId) (profile *models.NFProfile, err error) {
	if currentNssfContext == nil {
		return &models.NFProfile{}, fmt.Errorf("nssf context has not been initialized. NF profile cannot be built")
	}
	profile = models.NewNFProfileWithDefaults()
	profile.SetNfInstanceId(currentNssfContext.NfId)
	profile.SetNfType(models.NFTYPE_NSSF)
	profile.SetNfStatus(models.NFSTATUS_REGISTERED)
	if len(plmnConfig) > 0 {
		plmnCopy := make([]models.PlmnId, len(plmnConfig))
		copy(plmnCopy, plmnConfig)
		profile.SetPlmnList(plmnCopy)
	}
	profile.SetIpv4Addresses([]string{currentNssfContext.RegisterIPv4})
	var services []models.NFService
	for _, nfService := range currentNssfContext.NfService {
		services = append(services, nfService)
	}
	if len(services) > 0 {
		profile.SetNfServices(services)
	}
	return profile, err
}

var SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (prof *models.NFProfile, resourceNrfUri string, err error) {
	nssfSelf := nssfContext.NSSF_Self()
	nfProfile, err := getNfProfile(nssfSelf, plmnConfig)
	if err != nil {
		return &models.NFProfile{}, "", err
	}
	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = nssfSelf.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	apiClient := Nnrf_NFManagement.NewAPIClient(configuration)

	apiRegisterNFInstanceRequest := apiClient.NFInstanceIDDocumentAPI.RegisterNFInstance(context.TODO(), nfProfile.NfInstanceId)
	apiRegisterNFInstanceRequest = apiRegisterNFInstanceRequest.NFProfile(*nfProfile)
	receivedNfProfile, res, err := apiClient.NFInstanceIDDocumentAPI.RegisterNFInstanceExecute(apiRegisterNFInstanceRequest)
	if err != nil {
		return &models.NFProfile{}, "", err
	}
	if res == nil {
		return &models.NFProfile{}, "", fmt.Errorf("no response from server")
	}
	if res.Body != nil {
		defer func() {
			if bodyCloseErr := res.Body.Close(); bodyCloseErr != nil {
				logger.AppLog.Errorf("RegisterNFInstance response body cannot close: %+v", bodyCloseErr)
			}
		}()
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

var SendUpdateNFInstance = func(patchItem []models.PatchItem) (*models.NFProfile, *models.ProblemDetails, error) {
	logger.ConsumerLog.Debugln("send Update NFInstance")

	nssfSelf := nssfContext.NSSF_Self()
	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = nssfSelf.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	client := Nnrf_NFManagement.NewAPIClient(configuration)

	var res *http.Response
	apiUpdateNFInstanceRequest := client.NFInstanceIDDocumentAPI.UpdateNFInstance(context.Background(), nssfSelf.NfId)
	apiUpdateNFInstanceRequest = apiUpdateNFInstanceRequest.PatchItem(patchItem)
	receivedNfProfile, res, err := client.NFInstanceIDDocumentAPI.UpdateNFInstanceExecute(apiUpdateNFInstanceRequest)
	if res != nil && res.Body != nil {
		defer func() {
			if bodyCloseErr := res.Body.Close(); bodyCloseErr != nil {
				logger.AppLog.Errorf("UpdateNFInstance response body cannot close: %+v", bodyCloseErr)
			}
		}()
	}
	if err != nil {
		if openapiErr, ok := err.(openapi.GenericOpenAPIError); ok {
			if model := openapiErr.Model(); model != nil {
				if problem, ok := model.(models.ProblemDetails); ok {
					return &models.NFProfile{}, &problem, nil
				}
			}
		}
		return &models.NFProfile{}, nil, err
	}

	if res == nil {
		return &models.NFProfile{}, nil, fmt.Errorf("no response from server")
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		if receivedNfProfile == nil {
			return &models.NFProfile{}, nil, nil
		}
		return receivedNfProfile, nil, nil
	}
	return &models.NFProfile{}, nil, fmt.Errorf("unexpected response code")
}

var SendDeregisterNFInstance = func() error {
	logger.AppLog.Infoln("send Deregister NFInstance")

	nssfSelf := nssfContext.NSSF_Self()
	// Set client and set url
	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = nssfSelf.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	client := Nnrf_NFManagement.NewAPIClient(configuration)
	apiDeregisterNFInstanceRequest := client.NFInstanceIDDocumentAPI.DeregisterNFInstance(context.Background(), nssfSelf.NfId)
	res, err := client.NFInstanceIDDocumentAPI.DeregisterNFInstanceExecute(apiDeregisterNFInstanceRequest)
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
