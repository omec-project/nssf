// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2025 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package polling

import (
	"context"
	"reflect"
	"time"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/v2/models"
	"github.com/omec-project/openapi/v2/nfConfigApi"
	"github.com/omec-project/openapi/v2/nfconfigpoll"
)

const (
	initialPollingInterval = 5 * time.Second
	pollingPath            = "/nfconfig/plmn-snssai"
)

// StartPollingService initializes the polling service and starts it. The polling service
// continuously makes a HTTP GET request to the webconsole and updates the network configuration
func StartPollingService(ctx context.Context, webuiUri string, plmnConfigChan chan<- []models.PlmnId) {
	poller, err := newPoller(webuiUri, plmnConfigChan, initialPollingInterval)
	if err != nil {
		logger.PollConfigLog.Errorf("failed to start polling service: %v", err)
		return
	}
	poller.Run(ctx)
}

func newPoller(webuiUri string, plmnConfigChan chan<- []models.PlmnId, interval time.Duration) (*nfconfigpoll.Poller[[]nfConfigApi.PlmnSnssai], error) {
	handler := &plmnSnssaiHandler{plmnConfigChan: plmnConfigChan, currentPlmnConfig: []models.PlmnId{}}
	return nfconfigpoll.New(nfconfigpoll.Config[[]nfConfigApi.PlmnSnssai]{
		Name:     "PLMN-SNSSAI",
		Endpoint: webuiUri + pollingPath,
		Log:      logger.PollConfigLog,
		Interval: interval,
		Initial:  []nfConfigApi.PlmnSnssai{},
		OnChange: handler.handlePolledPlmnSnssaiConfig,
	})
}

// plmnSnssaiHandler applies a changed PLMN-SNSSAI config: it updates the supported NSSAI in the
// factory config and sends the PLMN list to the NF registration only when the PLMN list changed.
type plmnSnssaiHandler struct {
	plmnConfigChan    chan<- []models.PlmnId
	currentPlmnConfig []models.PlmnId
}

func (h *plmnSnssaiHandler) handlePolledPlmnSnssaiConfig(ctx context.Context, newPlmnSnssaiConfig []nfConfigApi.PlmnSnssai) error {
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	newPlmnConfig, newSupportedNssai := convertPlmnSnssaiList(newPlmnSnssaiConfig)

	if !reflect.DeepEqual(h.currentPlmnConfig, newPlmnConfig) {
		logger.PollConfigLog.Debugf("PLMN config changed %+v. Updating NF registration", newPlmnConfig)
		if err := nfconfigpoll.Send(ctx, h.plmnConfigChan, newPlmnConfig); err != nil {
			return err
		}
		h.currentPlmnConfig = newPlmnConfig
	}
	factory.NssfConfig.Configuration.SupportedNssaiInPlmnList = newSupportedNssai
	return nil
}

func convertPlmnSnssaiList(newConfig []nfConfigApi.PlmnSnssai) ([]models.PlmnId, factory.SupportedNssaiInPlmn) {
	newPlmnList := make([]models.PlmnId, 0, len(newConfig))
	newSupportedNssais := make(factory.SupportedNssaiInPlmn)

	for _, plmnSnssai := range newConfig {
		newPlmn := models.PlmnId{
			Mcc: plmnSnssai.PlmnId.GetMcc(),
			Mnc: plmnSnssai.PlmnId.GetMnc(),
		}
		newPlmnList = append(newPlmnList, newPlmn)
		newSnssaiSet := make(map[factory.SnssaiKey]struct{})
		for _, snssai := range plmnSnssai.SNssaiList {
			newSnssai := models.NewSnssai(snssai.GetSst())
			if snssai.GetSd() != "" {
				newSnssai.SetSd(snssai.GetSd())
			}
			newSnssaiSet[factory.SnssaiToKey(*newSnssai)] = struct{}{}
		}

		newSupportedNssais[newPlmn] = newSnssaiSet
	}

	return newPlmnList, newSupportedNssais
}
