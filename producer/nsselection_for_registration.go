// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF NS Selection
 *
 * NSSF Network Slice Selection Service
 */

package producer

import (
	"net/http"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/utils"
)

// Set Allowed NSSAI with Subscribed S-NSSAI(s) which are marked as default S-NSSAI(s)
func useDefaultSubscribedSnssai(
	param NsselectionQueryParameter, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	var mappingOfSnssai []models.MappingOfSnssai
	if param.HomePlmnId != nil {
		// Find mapping of Subscribed S-NSSAI of UE's HPLMN to S-NSSAI in Serving PLMN from NSSF configuration
		mappingOfSnssai = util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai == nil {
			logger.Nsselection.Warnf("no S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)
			return
		}
	}

	for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.GetSubscribedNssai() {
		if subscribedSnssai.GetDefaultIndication() {
			// Subscribed S-NSSAI is marked as default S-NSSAI

			var mappingOfSubscribedSnssai models.Snssai
			// TODO: Compared with Restricted S-NSSAI list in configuration under roaming scenario
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(subscribedSnssai.GetSubscribedSnssai()) {
				targetMapping, found := util.FindMappingWithHomeSnssai(subscribedSnssai.GetSubscribedSnssai(), mappingOfSnssai)

				if !found {
					logger.Nsselection.Warnf("no mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						subscribedSnssai.GetSubscribedSnssai(),
						*param.HomePlmnId)
					continue
				} else {
					mappingOfSubscribedSnssai = targetMapping.GetServingSnssai()
				}
			} else {
				mappingOfSubscribedSnssai = subscribedSnssai.GetSubscribedSnssai()
			}

			if param.Tai != nil && !util.CheckSupportedSnssaiInTa(mappingOfSubscribedSnssai, *param.Tai) {
				continue
			}

			var allowedSnssaiElement models.AllowedSnssai
			allowedSnssaiElement.SetAllowedSnssai(mappingOfSubscribedSnssai)
			nsiInformationList := util.GetNsiInformationListFromConfig(mappingOfSubscribedSnssai)
			if nsiInformationList != nil {
				// TODO: `NsiInformationList` should be slice in `AllowedSnssai` instead of pointer of slice
				allowedSnssaiElement.NsiInformationList = append(allowedSnssaiElement.NsiInformationList,
					nsiInformationList...)
			}
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(subscribedSnssai.GetSubscribedSnssai()) {
				allowedSnssaiElement.SetMappedHomeSnssai(subscribedSnssai.GetSubscribedSnssai())
			}

			// Default Access Type is set to 3GPP Access if no TAI is provided
			// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
			//       UE's Access Type could not be identified
			accessType := models.ACCESSTYPE__3_GPP_ACCESS
			if param.Tai != nil {
				accessType = util.GetAccessTypeFromConfig(*param.Tai)
			}

			util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
		}
	}
}

// Set Configured NSSAI with S-NSSAI(s) in Requested NSSAI which are marked as Default Configured NSSAI
func useDefaultConfiguredNssai(
	param NsselectionQueryParameter, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	for _, requestedSnssai := range param.SliceInfoRequestForRegistration.GetRequestedNssai() {
		// Check whether the Default Configured S-NSSAI is standard, which could be commonly decided by all roaming partners
		if !util.CheckStandardSnssai(requestedSnssai) {
			logger.Nsselection.Infof("s-nssai %+v in Requested NSSAI which based on Default Configured NSSAI is not standard",
				requestedSnssai)
			continue
		}

		// Check whether the Default Configured S-NSSAI is subscribed
		for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.GetSubscribedNssai() {
			if factory.SnssaiToKey(requestedSnssai) == factory.SnssaiToKey(subscribedSnssai.GetSubscribedSnssai()) {
				configuredSnssai := models.NewConfiguredSnssaiWithDefaults()
				configuredSnssai.SetConfiguredSnssai(requestedSnssai)

				authorizedNetworkSliceInfo.SetConfiguredNssai(append(authorizedNetworkSliceInfo.GetConfiguredNssai(), *configuredSnssai))
				break
			}
		}
	}
}

// Set Configured NSSAI with Subscribed S-NSSAI(s)
func setConfiguredNssai(
	param NsselectionQueryParameter, authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
) {
	var mappingOfSnssai []models.MappingOfSnssai
	if param.HomePlmnId != nil {
		// Find mapping of Subscribed S-NSSAI of UE's HPLMN to S-NSSAI in Serving PLMN from NSSF configuration
		mappingOfSnssai = util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai == nil {
			logger.Nsselection.Warnf("no S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)
			return
		}
	}

	for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.GetSubscribedNssai() {
		var mappingOfSubscribedSnssai models.Snssai
		if param.HomePlmnId != nil && !util.CheckStandardSnssai(subscribedSnssai.GetSubscribedSnssai()) {
			targetMapping, found := util.FindMappingWithHomeSnssai(subscribedSnssai.GetSubscribedSnssai(), mappingOfSnssai)

			if !found {
				logger.Nsselection.Warnf("no mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
					subscribedSnssai.GetSubscribedSnssai(),
					*param.HomePlmnId)
				continue
			} else {
				mappingOfSubscribedSnssai = targetMapping.GetServingSnssai()
			}
		} else {
			mappingOfSubscribedSnssai = subscribedSnssai.GetSubscribedSnssai()
		}

		if util.CheckSupportedSnssaiInPlmn(mappingOfSubscribedSnssai, param.Tai.GetPlmnId()) {
			configuredSnssai := models.NewConfiguredSnssaiWithDefaults()
			configuredSnssai.SetConfiguredSnssai(mappingOfSubscribedSnssai)
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(subscribedSnssai.GetSubscribedSnssai()) {
				configuredSnssai.SetMappedHomeSnssai(subscribedSnssai.GetSubscribedSnssai())
			}

			authorizedNetworkSliceInfo.SetConfiguredNssai(append(authorizedNetworkSliceInfo.GetConfiguredNssai(), *configuredSnssai))
		}
	}
}

// Network slice selection for registration
// The function is executed when the IE, `slice-info-request-for-registration`, is provided in query parameters
func nsselectionForRegistration(param NsselectionQueryParameter,
	authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
	problemDetails *models.ProblemDetails,
) int {
	var status int
	if param.HomePlmnId != nil {
		// Check whether UE's Home PLMN is supported when UE is a roamer
		if !util.CheckSupportedHplmn(*param.HomePlmnId) {
			rejectedNssaiInPlmn := append(authorizedNetworkSliceInfo.GetRejectedNssaiInPlmn(), param.SliceInfoRequestForRegistration.GetRequestedNssai()...)
			authorizedNetworkSliceInfo.SetRejectedNssaiInPlmn(rejectedNssaiInPlmn)

			status = http.StatusOK
			return status
		}
	}

	if param.Tai != nil {
		// Check whether UE's current TA is supported when UE provides TAI
		if !util.CheckSupportedTa(*param.Tai) {
			rejectedNssaiInTa := append(authorizedNetworkSliceInfo.GetRejectedNssaiInTa(), param.SliceInfoRequestForRegistration.GetRequestedNssai()...)
			authorizedNetworkSliceInfo.SetRejectedNssaiInTa(rejectedNssaiInTa)

			status = http.StatusOK
			return status
		}
	}

	if param.SliceInfoRequestForRegistration.GetRequestMapping() {
		// Based on TS 29.531 v15.2.0, when `requestMapping` is set to true, the NSSF shall return the VPLMN specific
		// mapped S-NSSAI values for the S-NSSAI values in `subscribedNssai`. But also `sNssaiForMapping` shall be
		// provided if `requestMapping` is set to true. In the implementation, the NSSF would return mapped S-NSSAIs
		// for S-NSSAIs in both `sNssaiForMapping` and `subscribedSnssai` if present

		if param.HomePlmnId == nil {
			problemDetail := "[Query Parameter] `home-plmn-id` should be provided when requesting VPLMN specific mapped S-NSSAI values"
			invalidParam := models.NewInvalidParam("home-plmn-id")
			invalidParam.SetReason(problemDetail)
			invalidParams := []models.InvalidParam{*invalidParam}
			*problemDetails = *utils.ProblemDetailsWithInvalidParams(util.INVALID_REQUEST, http.StatusBadRequest, problemDetail, invalidParams)
			status = http.StatusBadRequest
			return status
		}

		mappingOfSnssai := util.GetMappingOfPlmnFromConfig(*param.HomePlmnId)

		if mappingOfSnssai != nil {
			// Find mappings for S-NSSAIs in `subscribedSnssai`
			for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.GetSubscribedNssai() {
				if util.CheckStandardSnssai(subscribedSnssai.GetSubscribedSnssai()) {
					continue
				}

				targetMapping, found := util.FindMappingWithHomeSnssai(subscribedSnssai.GetSubscribedSnssai(), mappingOfSnssai)

				if !found {
					logger.Nsselection.Warnf("no mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						subscribedSnssai.GetSubscribedSnssai(),
						*param.HomePlmnId)
					continue
				} else {
					// Add mappings to Allowed NSSAI list
					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = targetMapping.GetServingSnssai()
					mappedHomeSnssai := subscribedSnssai.GetSubscribedSnssai()
					allowedSnssaiElement.MappedHomeSnssai = &mappedHomeSnssai

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.ACCESSTYPE__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
				}
			}

			// Find mappings for S-NSSAIs in `sNssaiForMapping`
			for _, snssai := range param.SliceInfoRequestForRegistration.GetSNssaiForMapping() {
				if util.CheckStandardSnssai(snssai) {
					continue
				}

				targetMapping, found := util.FindMappingWithHomeSnssai(snssai, mappingOfSnssai)

				if !found {
					logger.Nsselection.Warnf("No mapping of Subscribed S-NSSAI %+v in PLMN %+v in NSSF configuration",
						snssai,
						*param.HomePlmnId)
					continue
				} else {
					// Add mappings to Allowed NSSAI list
					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = targetMapping.GetServingSnssai()
					snssaiCopy := snssai
					allowedSnssaiElement.MappedHomeSnssai = &snssaiCopy

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.ACCESSTYPE__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)
				}
			}

			status = http.StatusOK
			return status
		} else {
			logger.Nsselection.Warnf("no S-NSSAI mapping of UE's HPLMN %+v in NSSF configuration", *param.HomePlmnId)

			status = http.StatusOK
			return status
		}
	}

	checkInvalidRequestedNssai := false
	if len(param.SliceInfoRequestForRegistration.GetRequestedNssai()) != 0 {
		// Requested NSSAI is provided
		// Verify which S-NSSAI(s) in the Requested NSSAI are permitted based on comparing the Subscribed S-NSSAI(s)

		if param.Tai != nil &&
			!util.CheckSupportedNssaiInPlmn(param.SliceInfoRequestForRegistration.GetRequestedNssai(), param.Tai.GetPlmnId()) {
			// Return ProblemDetails indicating S-NSSAI is not supported
			// TODO: Based on TS 23.501 V15.2.0, if the Requested NSSAI includes an S-NSSAI that is not valid in the
			//       Serving PLMN, the NSSF may derive the Configured NSSAI for Serving PLMN
			*problemDetails = *utils.ProblemDetails(
				util.UNSUPPORTED_RESOURCE,
				http.StatusForbidden,
				"S-NSSAI in Requested NSSAI is not supported in PLMN",
			)
			problemDetails.SetCause(utils.CauseSnssaiNotSupported)
			status = http.StatusForbidden
			return status
		}

		// Check if any Requested S-NSSAIs is present in Subscribed S-NSSAIs
		checkIfRequestAllowed := false

		for _, requestedSnssai := range param.SliceInfoRequestForRegistration.GetRequestedNssai() {
			if param.Tai != nil && !util.CheckSupportedSnssaiInTa(requestedSnssai, *param.Tai) {
				// Requested S-NSSAI does not supported in UE's current TA
				// Add it to Rejected NSSAI in TA
				rejectedNssaiInTa := append(authorizedNetworkSliceInfo.GetRejectedNssaiInTa(), requestedSnssai)
				authorizedNetworkSliceInfo.SetRejectedNssaiInTa(rejectedNssaiInTa)
				continue
			}

			var mappingOfRequestedSnssai models.Snssai
			// TODO: Compared with Restricted S-NSSAI list in configuration under roaming scenario
			if param.HomePlmnId != nil && !util.CheckStandardSnssai(requestedSnssai) {
				// Standard S-NSSAIs are supported to be commonly decided by all roaming partners
				// Only non-standard S-NSSAIs are required to find mappings
				targetMapping, found := util.FindMappingWithServingSnssai(requestedSnssai,
					param.SliceInfoRequestForRegistration.GetMappingOfNssai())

				if !found {
					// No mapping of Requested S-NSSAI to HPLMN S-NSSAI is provided by UE
					// TODO: Search for local configuration if there is no provided mapping from UE, and update UE's
					//       Configured NSSAI
					checkInvalidRequestedNssai = true
					rejectedNssaiInPlmn := append(authorizedNetworkSliceInfo.GetRejectedNssaiInPlmn(), requestedSnssai)
					authorizedNetworkSliceInfo.SetRejectedNssaiInPlmn(rejectedNssaiInPlmn)
					continue
				} else {
					// TODO: Check if mappings of S-NSSAIs are correct
					//       If not, update UE's Configured NSSAI
					mappingOfRequestedSnssai = targetMapping.GetHomeSnssai()
				}
			} else {
				mappingOfRequestedSnssai = requestedSnssai
			}

			hitSubscription := false
			for _, subscribedSnssai := range param.SliceInfoRequestForRegistration.GetSubscribedNssai() {
				if factory.SnssaiToKey(mappingOfRequestedSnssai) == factory.SnssaiToKey(subscribedSnssai.GetSubscribedSnssai()) {
					// Requested S-NSSAI matches one of Subscribed S-NSSAI
					// Add it to Allowed NSSAI list
					hitSubscription = true

					var allowedSnssaiElement models.AllowedSnssai
					allowedSnssaiElement.AllowedSnssai = requestedSnssai
					nsiInformationList := util.GetNsiInformationListFromConfig(requestedSnssai)
					if nsiInformationList != nil {
						// TODO: `NsiInformationList` should be slice in `AllowedSnssai` instead of pointer of slice
						allowedSnssaiElement.NsiInformationList = append(allowedSnssaiElement.NsiInformationList,
							nsiInformationList...)
					}
					if param.HomePlmnId != nil && !util.CheckStandardSnssai(requestedSnssai) {
						mappedHomeSnssai := subscribedSnssai.GetSubscribedSnssai()
						allowedSnssaiElement.MappedHomeSnssai = &mappedHomeSnssai
					}

					// Default Access Type is set to 3GPP Access if no TAI is provided
					// TODO: Depend on operator implementation, it may also return S-NSSAIs in all valid Access Type if
					//       UE's Access Type could not be identified
					accessType := models.ACCESSTYPE__3_GPP_ACCESS
					if param.Tai != nil {
						accessType = util.GetAccessTypeFromConfig(*param.Tai)
					}

					util.AddAllowedSnssai(allowedSnssaiElement, accessType, authorizedNetworkSliceInfo)

					checkIfRequestAllowed = true
					break
				}
			}

			if !hitSubscription {
				// Requested S-NSSAI does not match any Subscribed S-NSSAI
				// Add it to Rejected NSSAI in PLMN
				checkInvalidRequestedNssai = true
				rejectedNssaiInPlmn := append(authorizedNetworkSliceInfo.GetRejectedNssaiInPlmn(), requestedSnssai)
				authorizedNetworkSliceInfo.SetRejectedNssaiInPlmn(rejectedNssaiInPlmn)
			}
		}

		if !checkIfRequestAllowed {
			// No S-NSSAI from Requested NSSAI is present in Subscribed S-NSSAIs
			// Subscribed S-NSSAIs marked as default are used
			useDefaultSubscribedSnssai(param, authorizedNetworkSliceInfo)
		}
	} else {
		// No Requested NSSAI is provided
		// Subscribed S-NSSAIs marked as default are used
		checkInvalidRequestedNssai = true
		useDefaultSubscribedSnssai(param, authorizedNetworkSliceInfo)
	}

	if param.Tai != nil &&
		!util.CheckAllowedNssaiInAmfTa(authorizedNetworkSliceInfo.AllowedNssaiList, param.NfId, *param.Tai) {
		util.AddAmfInformation(*param.Tai, authorizedNetworkSliceInfo)
	}

	if param.SliceInfoRequestForRegistration.GetDefaultConfiguredSnssaiInd() {
		// Default Configured NSSAI Indication is received from AMF
		// Determine the Configured NSSAI based on the Default Configured NSSAI
		useDefaultConfiguredNssai(param, authorizedNetworkSliceInfo)
	} else if checkInvalidRequestedNssai {
		// No Requested NSSAI is provided or the Requested NSSAI includes an S-NSSAI that is not valid
		// Determine the Configured NSSAI based on the subscription
		// Configure available NSSAI for UE in its PLMN
		// If TAI is not provided, then unable to check if S-NSSAIs is supported in the PLMN
		if param.Tai != nil {
			setConfiguredNssai(param, authorizedNetworkSliceInfo)
		}
	}

	status = http.StatusOK
	return status
}
