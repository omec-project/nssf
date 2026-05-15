// Copyright (c) 2026 Intel Corporation
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"github.com/omec-project/openapi/v2/models"
)

type NsselectionQueryParameter struct {
	NfType                          *models.NFType                   `json:"nf-type"`
	NfId                            string                           `json:"nf-id"`
	SliceInfoRequestForRegistration *models.SliceInfoForRegistration `json:"slice-info-request-for-registration,omitempty"`
	SliceInfoRequestForPduSession   *models.SliceInfoForPDUSession   `json:"slice-info-request-for-pdu-session,omitempty"`
	HomePlmnId                      *models.PlmnId                   `json:"home-plmn-id,omitempty"`
	Tai                             *models.Tai                      `json:"tai,omitempty"`
	SupportedFeatures               string                           `json:"supported-features,omitempty"`
}
