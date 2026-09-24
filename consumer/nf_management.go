// SPDX-FileCopyrightText: 2025 Canonical Ltd
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package consumer

import (
	"context"
	"net/http"
	"strings"

	nssfContext "github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/Nnrf_NFManagement"
	"github.com/omec-project/openapi/v2/models"
)

const errServerNoResponse = "no response from server"

func closeNFManagementResponseBody(res *http.Response, operation string) {
	if res == nil || res.Body == nil {
		return
	}
	if bodyCloseErr := res.Body.Close(); bodyCloseErr != nil {
		logger.ConsumerLog.Errorf("%s response body cannot close: %+v", operation, bodyCloseErr)
	}
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

var SendRegisterNFInstance = func(plmnConfig []models.PlmnId) (prof *models.NFProfile, resourceNrfUri string, err error) {
	self := nssfContext.NSSF_Self()
	nfProfile, err := getNfProfile(self, plmnConfig)
	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}

	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = self.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	client := Nnrf_NFManagement.NewAPIClient(configuration)
	apiRegisterNFInstanceRequest := client.NFInstanceIDDocumentAPI.RegisterNFInstance(context.TODO(), nfProfile.GetNfInstanceId())
	apiRegisterNFInstanceRequest = apiRegisterNFInstanceRequest.NFProfile(nfProfile)
	receivedNfProfile, res, err := client.NFInstanceIDDocumentAPI.RegisterNFInstanceExecute(apiRegisterNFInstanceRequest)
	defer closeNFManagementResponseBody(res, "RegisterNFInstance")
	logger.ConsumerLog.Debugf("registering NF Instance using profile: %+v", nfProfile)

	if err != nil {
		return models.NewNFProfileWithDefaults(), "", err
	}
	if res == nil {
		return models.NewNFProfileWithDefaults(), "", openapi.ReportError(errServerNoResponse)
	}

	switch res.StatusCode {
	case http.StatusOK: // NFUpdate
		logger.ConsumerLog.Debugln("NSSF NF profile updated with complete replacement")
		return receivedNfProfile, "", nil
	case http.StatusCreated: // NFRegister
		resourceUri := res.Header.Get("Location")
		resourceNrfUri = resourceUri[:strings.Index(resourceUri, "/nnrf-nfm/")]
		retrieveNfInstanceId := resourceUri[strings.LastIndex(resourceUri, "/")+1:]
		self.NfId = retrieveNfInstanceId
		logger.ConsumerLog.Debugln("NSSF NF profile registered to the NRF")
		return receivedNfProfile, resourceNrfUri, nil
	default:
		return receivedNfProfile, "", openapi.ReportError("NRF returned unexpected status code %d", res.StatusCode)
	}
}

var SendDeregisterNFInstance = func() error {
	logger.ConsumerLog.Infoln("send Deregister NFInstance")

	self := nssfContext.NSSF_Self()
	// Set client and set url
	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = self.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	client := Nnrf_NFManagement.NewAPIClient(configuration)
	apiDeregisterNFInstanceRequest := client.NFInstanceIDDocumentAPI.DeregisterNFInstance(context.Background(), self.NfId)
	res, err := client.NFInstanceIDDocumentAPI.DeregisterNFInstanceExecute(apiDeregisterNFInstanceRequest)
	defer closeNFManagementResponseBody(res, "DeregisterNFInstance")
	if err != nil {
		return err
	}
	if res == nil {
		return openapi.ReportError(errServerNoResponse)
	}
	if res.StatusCode == http.StatusNoContent {
		return nil
	}
	return openapi.ReportError("unexpected response code")
}

var SendUpdateNFInstance = func(patchItem []models.PatchItem) (receivedNfProfile *models.NFProfile, problemDetails *models.ProblemDetails, err error) {
	logger.ConsumerLog.Debugln("send Update NFInstance")

	self := nssfContext.NSSF_Self()
	configuration := Nnrf_NFManagement.NewConfiguration()
	serverConfig := &configuration.Servers[0]
	if apiRootVar, exists := serverConfig.Variables["apiRoot"]; exists {
		apiRootVar.DefaultValue = self.NrfUri
		serverConfig.Variables["apiRoot"] = apiRootVar
	}
	client := Nnrf_NFManagement.NewAPIClient(configuration)

	var res *http.Response
	apiUpdateNFInstanceRequest := client.NFInstanceIDDocumentAPI.UpdateNFInstance(context.Background(), self.NfId)
	apiUpdateNFInstanceRequest = apiUpdateNFInstanceRequest.PatchItem(patchItem)
	receivedNfProfile, res, err = client.NFInstanceIDDocumentAPI.UpdateNFInstanceExecute(apiUpdateNFInstanceRequest)
	defer closeNFManagementResponseBody(res, "UpdateNFInstance")
	if err != nil {
		if openapiErr, ok := openapi.AsGenericOpenAPIError(err); ok {
			if model := openapiErr.Model(); model != nil {
				if problem, ok := model.(models.ProblemDetails); ok {
					return models.NewNFProfileWithDefaults(), &problem, nil
				}
			}
		}
		return models.NewNFProfileWithDefaults(), nil, err
	}

	if res == nil {
		return models.NewNFProfileWithDefaults(), nil, openapi.ReportError(errServerNoResponse)
	}
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		return receivedNfProfile, nil, nil
	}
	return models.NewNFProfileWithDefaults(), nil, openapi.ReportError("unexpected response code")
}
