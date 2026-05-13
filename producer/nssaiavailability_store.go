// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF NSSAI Availability
 *
 * NSSF NSSAI Availability Service
 */

package producer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/plugin"
	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/utils"
)

// NSSAIAvailability DELETE method
func NSSAIAvailabilityDeleteProcedure(nfId string) *models.ProblemDetails {
	var problemDetails *models.ProblemDetails
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	for i, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfId {
			factory.NssfConfig.Configuration.AmfList = append(
				factory.NssfConfig.Configuration.AmfList[:i],
				factory.NssfConfig.Configuration.AmfList[i+1:]...)
			return nil
		}
	}

	problemDetails = models.NewProblemDetails()
	problemDetails.SetTitle(util.UNSUPPORTED_RESOURCE)
	problemDetails.SetStatus(http.StatusNotFound)
	problemDetails.SetDetail(fmt.Sprintf("AMF ID '%s' does not exist", nfId))
	return problemDetails
}

// NSSAIAvailability PATCH method
func NSSAIAvailabilityPatchProcedure(nssaiAvailabilityUpdateInfo plugin.PatchDocument, nfId string) (
	*models.AuthorizedNssaiAvailabilityInfo, *models.ProblemDetails,
) {
	response := models.NewAuthorizedNssaiAvailabilityInfoWithDefaults()
	problemDetails := models.NewProblemDetails()

	var amfIdx int
	var original []byte
	hitAmf := false
	factory.ConfigLock.RLock()
	for amfIdx, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfId {
			// Since json-patch package does not have idea of optional field of datatype,
			// provide with null or empty value instead of omitting the field
			var temp []models.SupportedNssaiAvailabilityData
			configData, err := json.Marshal(factory.NssfConfig.Configuration.AmfList[amfIdx].SupportedNssaiAvailabilityData)
			if err != nil {
				factory.ConfigLock.RUnlock()
				logger.Nssaiavailability.Errorf("marshal error in NSSAIAvailabilityPatchProcedure: %+v", err)
				return nil, utils.ProblemDetailsSystemFailure(err.Error())
			}
			if err = json.Unmarshal(configData, &temp); err != nil {
				factory.ConfigLock.RUnlock()
				logger.Nssaiavailability.Errorf("unmarshal error in NSSAIAvailabilityPatchProcedure: %+v", err)
				return nil, utils.ProblemDetailsSystemFailure(err.Error())
			}
			const dummyString string = "DUMMY"
			for i := range temp {
				for j := range temp[i].SupportedSnssaiList {
					if temp[i].SupportedSnssaiList[j].GetSd() == "" {
						temp[i].SupportedSnssaiList[j].SetSd(dummyString)
					}
				}
			}
			original, err = json.Marshal(temp)
			if err != nil {
				logger.Nssaiavailability.Errorf("marshal error in NSSAIAvailabilityPatchProcedure: %+v", err)
				factory.ConfigLock.RUnlock()
				return nil, utils.ProblemDetailsSystemFailure(err.Error())
			}
			original = bytes.ReplaceAll(original, []byte(dummyString), []byte(""))

			// original, _ = json.Marshal(factory.NssfConfig.Configuration.AmfList[amfIdx].SupportedNssaiAvailabilityData)

			hitAmf = true
			break
		}
	}
	factory.ConfigLock.RUnlock()
	if !hitAmf {
		problemDetails.SetTitle(util.UNSUPPORTED_RESOURCE)
		problemDetails.SetStatus(http.StatusNotFound)
		problemDetails.SetDetail(fmt.Sprintf("AMF ID '%s' does not exist", nfId))
		return nil, problemDetails
	}

	// TODO: Check if returned HTTP status codes or problem details are proper when errors occur

	// Provide JSON string with null or empty value in `Value` of `PatchItem`
	for i, patchItem := range nssaiAvailabilityUpdateInfo {
		if reflect.ValueOf(patchItem.Value).Kind() == reflect.Map {
			_, exist := patchItem.Value.(map[string]interface{})["sst"]
			_, notExist := patchItem.Value.(map[string]interface{})["sd"]
			if exist && !notExist {
				nssaiAvailabilityUpdateInfo[i].Value.(map[string]interface{})["sd"] = ""
			}
		}
	}
	patchJSON, err := json.Marshal(nssaiAvailabilityUpdateInfo)
	if err != nil {
		logger.Nssaiavailability.Errorf("marshal error in NSSAIAvailabilityPatchProcedure: %+v", err)
	}

	patch, err := jsonpatch.DecodePatch(patchJSON)
	if err != nil {
		problemDetails = utils.ProblemDetailsMalformedRequestSyntax(err.Error())
		return nil, problemDetails
	}

	modified, err := patch.Apply(original)
	if err != nil {
		problemDetails.SetTitle(util.INVALID_REQUEST)
		problemDetails.SetStatus(http.StatusConflict)
		problemDetails.SetDetail(err.Error())
		return nil, problemDetails
	}

	factory.ConfigLock.Lock()
	err = json.Unmarshal(modified, &factory.NssfConfig.Configuration.AmfList[amfIdx].SupportedNssaiAvailabilityData)
	factory.ConfigLock.Unlock()
	if err != nil {
		problemDetails.SetTitle(util.INVALID_REQUEST)
		problemDetails.SetStatus(http.StatusBadRequest)
		problemDetails.SetDetail(err.Error())
		return nil, problemDetails
	}

	// Return all authorized NSSAI availability information
	response.AuthorizedNssaiAvailabilityData, err = util.AuthorizeOfAmfFromConfig(nfId)
	if err != nil {
		logger.Nssaiavailability.Errorf("util AuthorizeOfAmfFromConfig error in NSSAIAvailabilityPatchProcedure: %+v", err)
	}

	// TODO: Return authorized NSSAI availability information of updated TAI only

	return response, nil
}

// NSSAIAvailability PUT method
func NSSAIAvailabilityPutProcedure(nssaiAvailabilityInfo models.NssaiAvailabilityInfo, nfId string) (
	*models.AuthorizedNssaiAvailabilityInfo, *models.ProblemDetails,
) {
	response := models.NewAuthorizedNssaiAvailabilityInfoWithDefaults()

	for _, s := range nssaiAvailabilityInfo.SupportedNssaiAvailabilityData {
		if !util.CheckSupportedNssaiInPlmn(s.SupportedSnssaiList, s.Tai.PlmnId) {
			problemDetails := models.NewProblemDetails()
			problemDetails.SetTitle(util.UNSUPPORTED_RESOURCE)
			problemDetails.SetStatus(http.StatusForbidden)
			problemDetails.SetDetail("S-NSSAI in Requested NSSAI is not supported in PLMN")
			problemDetails.SetCause("SNSSAI_NOT_SUPPORTED")
			return nil, problemDetails
		}
	}

	// TODO: Currently authorize all the provided S-NSSAIs
	//       Take some issue into consideration e.g. operator policies

	hitAmf := false
	// Find AMF configuration of given NfId
	// If found, then update the SupportedNssaiAvailabilityData
	factory.ConfigLock.Lock()
	for i, amfConfig := range factory.NssfConfig.Configuration.AmfList {
		if amfConfig.NfId == nfId {
			factory.NssfConfig.Configuration.AmfList[i].SupportedNssaiAvailabilityData = nssaiAvailabilityInfo.SupportedNssaiAvailabilityData

			hitAmf = true
			break
		}
	}
	factory.ConfigLock.Unlock()

	// If no AMF record is found, create a new one
	if !hitAmf {
		var amfConfig factory.AmfConfig
		amfConfig.NfId = nfId
		amfConfig.SupportedNssaiAvailabilityData = nssaiAvailabilityInfo.SupportedNssaiAvailabilityData
		factory.ConfigLock.Lock()
		factory.NssfConfig.Configuration.AmfList = append(factory.NssfConfig.Configuration.AmfList, amfConfig)
		factory.ConfigLock.Unlock()
	}

	// Return all authorized NSSAI availability information
	// a.AuthorizedNssaiAvailabilityData, _ = authorizeOfAmfFromConfig(nfId)

	// Return authorized NSSAI availability information of updated TAI only
	for _, s := range nssaiAvailabilityInfo.SupportedNssaiAvailabilityData {
		authorizedNssaiAvailabilityData, err := util.AuthorizeOfAmfTaFromConfig(nfId, s.Tai)
		if err == nil {
			response.AuthorizedNssaiAvailabilityData = append(response.AuthorizedNssaiAvailabilityData, authorizedNssaiAvailabilityData)
		} else {
			logger.Nssaiavailability.Warnln(err.Error())
		}
	}

	return response, nil
}
