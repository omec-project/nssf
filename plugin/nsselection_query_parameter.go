// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/*
 * NSSF Plugin
 */

package plugin

import (
	. "github.com/free5gc/openapi/models"
)

type NsselectionQueryParameter struct {
	NfType *NfType `json:"nf-type"`

	NfId string `json:"nf-id"`

	SliceInfoRequestForRegistration *SliceInfoForRegistration `json:"slice-info-request-for-registration,omitempty"`

	SliceInfoRequestForPduSession *SliceInfoForPduSession `json:"slice-info-request-for-pdu-session,omitempty"`

	HomePlmnId *PlmnId `json:"home-plmn-id,omitempty"`

	Tai *Tai `json:"tai,omitempty"`

	SupportedFeatures string `json:"supported-features,omitempty"`
}
