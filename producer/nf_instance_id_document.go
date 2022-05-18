// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package producer

import (
	"net/http"

	"github.com/omec-project/http_wrapper"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/plugin"
	"github.com/omec-project/openapi/models"
)

// HandleNSSAIAvailabilityDelete - Deletes an already existing S-NSSAIs per TA
// provided by the NF service consumer (e.g AMF)
func HandleNSSAIAvailabilityDelete(request *http_wrapper.Request) *http_wrapper.Response {
	logger.Nssaiavailability.Infof("Handle NSSAIAvailabilityDelete")

	nfID := request.Params["nfId"]

	problemDetails := NSSAIAvailabilityDeleteProcedure(nfID)

	if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	return http_wrapper.NewResponse(http.StatusNoContent, nil, nil)
}

// HandleNSSAIAvailabilityPatch - Updates an already existing S-NSSAIs per TA
// provided by the NF service consumer (e.g AMF)
func HandleNSSAIAvailabilityPatch(request *http_wrapper.Request) *http_wrapper.Response {
	logger.Nssaiavailability.Infof("Handle NSSAIAvailabilityPatch")

	nssaiAvailabilityUpdateInfo := request.Body.(plugin.PatchDocument)
	nfID := request.Params["nfId"]

	// TODO: Request NfProfile of NfId from NRF
	//       Check if NfId is valid AMF and obtain AMF Set ID
	//       If NfId is invalid, return ProblemDetails with code 404 Not Found
	//       If NF consumer is not authorized to update NSSAI availability, return ProblemDetails with code 403 Forbidden

	response, problemDetails := NSSAIAvailabilityPatchProcedure(nssaiAvailabilityUpdateInfo, nfID)

	if response != nil {
		return http_wrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return http_wrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}

// HandleNSSAIAvailabilityPut - Updates/replaces the NSSF
// with the S-NSSAIs the NF service consumer (e.g AMF) supports per TA
func HandleNSSAIAvailabilityPut(request *http_wrapper.Request) *http_wrapper.Response {
	logger.Nssaiavailability.Infof("Handle NSSAIAvailabilityPut")

	nssaiAvailabilityInfo := request.Body.(models.NssaiAvailabilityInfo)
	nfID := request.Params["nfId"]

	response, problemDetails := NSSAIAvailabilityPutProcedure(nssaiAvailabilityInfo, nfID)

	if response != nil {
		return http_wrapper.NewResponse(http.StatusOK, nil, response)
	} else if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	}
	problemDetails = &models.ProblemDetails{
		Status: http.StatusForbidden,
		Cause:  "UNSPECIFIED",
	}
	return http_wrapper.NewResponse(http.StatusForbidden, nil, problemDetails)
}
