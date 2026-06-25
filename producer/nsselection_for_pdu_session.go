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
	"fmt"
	"math/rand"
	"net/http"

	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/utils"
)

func selectNsiInformation(nsiInformationList []models.NsiInformation) models.NsiInformation {
	// TODO: Algorithm to select Network Slice Instance
	//       Take roaming indication into consideration

	// Randomly select a Network Slice Instance
	idx := rand.Intn(len(nsiInformationList))
	return nsiInformationList[idx]
}

// Network slice selection for PDU session
// The function is executed when the IE, `slice-info-for-pdu-session`, is provided in query parameters
func nsselectionForPduSession(param NsselectionQueryParameter,
	authorizedNetworkSliceInfo *models.AuthorizedNetworkSliceInfo,
	problemDetails *models.ProblemDetails,
) int {
	var status int
	if param.HomePlmnId != nil {
		// Check whether UE's Home PLMN is supported when UE is a roamer
		if !util.CheckSupportedHplmn(*param.HomePlmnId) {
			rejectedNssaiInPlmn := append(authorizedNetworkSliceInfo.GetRejectedNssaiInPlmn(), param.SliceInfoRequestForPduSession.GetSNssai())
			authorizedNetworkSliceInfo.SetRejectedNssaiInPlmn(rejectedNssaiInPlmn)

			status = http.StatusOK
			return status
		}
	}

	if param.Tai != nil {
		// Check whether UE's current TA is supported when UE provides TAI
		if !util.CheckSupportedTa(*param.Tai) {
			rejectedNssaiInTa := append(authorizedNetworkSliceInfo.GetRejectedNssaiInTa(), param.SliceInfoRequestForPduSession.GetSNssai())
			authorizedNetworkSliceInfo.SetRejectedNssaiInTa(rejectedNssaiInTa)

			status = http.StatusOK
			return status
		}
	}

	if param.Tai != nil &&
		!util.CheckSupportedSnssaiInPlmn(param.SliceInfoRequestForPduSession.GetSNssai(), param.Tai.GetPlmnId()) {
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

	if param.HomePlmnId != nil {
		if param.SliceInfoRequestForPduSession.GetRoamingIndication() == models.ROAMINGINDICATION_NON_ROAMING {
			problemDetail := "`home-plmn-id` is provided, which contradicts `roamingIndication`:'NON_ROAMING'"
			invalidParam := models.NewInvalidParam("home-plmn-id")
			invalidParam.SetReason(problemDetail)
			invalidParams := []models.InvalidParam{*invalidParam}
			*problemDetails = *utils.ProblemDetailsWithInvalidParams(util.INVALID_REQUEST, http.StatusBadRequest, problemDetail, invalidParams)
			status = http.StatusBadRequest
			return status
		}
	} else {
		if param.SliceInfoRequestForPduSession.GetRoamingIndication() != models.ROAMINGINDICATION_NON_ROAMING {
			problemDetail := fmt.Sprintf("`home-plmn-id` is not provided, which contradicts `roamingIndication`:'%s'",
				string(param.SliceInfoRequestForPduSession.GetRoamingIndication()))
			invalidParam := models.NewInvalidParam("home-plmn-id")
			invalidParam.SetReason(problemDetail)
			invalidParams := []models.InvalidParam{*invalidParam}
			*problemDetails = *utils.ProblemDetailsWithInvalidParams(util.INVALID_REQUEST, http.StatusBadRequest, problemDetail, invalidParams)
			status = http.StatusBadRequest
			return status
		}
	}

	if param.Tai != nil && !util.CheckSupportedSnssaiInTa(param.SliceInfoRequestForPduSession.GetSNssai(), *param.Tai) {
		// Requested S-NSSAI does not supported in UE's current TA
		// Add it to Rejected NSSAI in TA
		rejectedNssaiInTa := append(authorizedNetworkSliceInfo.GetRejectedNssaiInTa(), param.SliceInfoRequestForPduSession.GetSNssai())
		authorizedNetworkSliceInfo.SetRejectedNssaiInTa(rejectedNssaiInTa)
		status = http.StatusOK
		return status
	}

	nsiInformationList := util.GetNsiInformationListFromConfig(param.SliceInfoRequestForPduSession.GetSNssai())

	if nsiInformationList != nil {
		nsiInformation := selectNsiInformation(nsiInformationList)
		authorizedNetworkSliceInfo.SetNsiInformation(nsiInformation)
	}

	return http.StatusOK
}
