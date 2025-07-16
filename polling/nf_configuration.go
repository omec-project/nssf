// SPDX-FileCopyrightText: 2025 Canonical Ltd

// SPDX-License-Identifier: Apache-2.0
//

package polling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/openapi/nfConfigApi"
)

const (
	initialPollingInterval = 5 * time.Second
	pollingMaxBackoff      = 40 * time.Second
	pollingBackoffFactor   = 2
	pollingPath            = "/nfconfig/plmn-snssai"
)

type nfConfigPoller struct {
	plmnConfigChan          chan<- []models.PlmnId
	currentPlmnSnssaiConfig []nfConfigApi.PlmnSnssai
	currentPlmnConfig       []models.PlmnId
	client                  *http.Client
}

// StartPollingService initializes the polling service and starts it. The polling service
// continuously makes a HTTP GET request to the webconsole and updates the network configuration
func StartPollingService(ctx context.Context, webuiUri string, plmnConfigChan chan<- []models.PlmnId) {
	poller := nfConfigPoller{
		plmnConfigChan:          plmnConfigChan,
		currentPlmnSnssaiConfig: []nfConfigApi.PlmnSnssai{},
		currentPlmnConfig:       []models.PlmnId{},
		client:                  &http.Client{Timeout: initialPollingInterval},
	}
	interval := initialPollingInterval
	pollingEndpoint := webuiUri + pollingPath
	logger.PollConfigLog.Infof("Started polling service on %s every %v", pollingEndpoint, initialPollingInterval)
	for {
		select {
		case <-ctx.Done():
			logger.PollConfigLog.Infoln("Polling service shutting down")
			return
		case <-time.After(interval):
			newConfig, err := fetchPlmnConfig(&poller, pollingEndpoint)
			if err != nil {
				interval = minDuration(interval*time.Duration(pollingBackoffFactor), pollingMaxBackoff)
				logger.PollConfigLog.Errorf("Polling error. Retrying in %v: %+v", interval, err)
				continue
			}
			interval = initialPollingInterval
			poller.handlePolledPlmnSnssaiConfig(newConfig)
		}
	}
}

var fetchPlmnConfig = func(p *nfConfigPoller, endpoint string) ([]nfConfigApi.PlmnSnssai, error) {
	return p.fetchPlmnConfig(endpoint)
}

func (p *nfConfigPoller) fetchPlmnConfig(pollingEndpoint string) ([]nfConfigApi.PlmnSnssai, error) {
	ctx, cancel := context.WithTimeout(context.Background(), initialPollingInterval)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollingEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET %v failed: %w", pollingEndpoint, err)
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, fmt.Errorf("unexpected Content-Type: got %s, want application/json", contentType)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var config []nfConfigApi.PlmnSnssai
		if err := json.Unmarshal(body, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		return config, nil

	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, fmt.Errorf("server returned %d error code", resp.StatusCode)
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (p *nfConfigPoller) handlePolledPlmnSnssaiConfig(newPlmnSnssaiConfig []nfConfigApi.PlmnSnssai) {
	if reflect.DeepEqual(p.currentPlmnSnssaiConfig, newPlmnSnssaiConfig) {
		logger.PollConfigLog.Debugf("PLMN-SNSSAI config did not change %+v", p.currentPlmnSnssaiConfig)
		return
	}
	factory.ConfigLock.Lock()
	defer factory.ConfigLock.Unlock()
	p.currentPlmnSnssaiConfig = newPlmnSnssaiConfig
	logger.PollConfigLog.Infof("PLMN-SNSSAI config changed. New PLMN-SNSSAI config: %+v", p.currentPlmnSnssaiConfig)
	newPlmnConfig, newSupportedNssai := convertPlmnSnssaiList(p.currentPlmnSnssaiConfig)

	if !reflect.DeepEqual(p.currentPlmnConfig, newPlmnConfig) {
		logger.PollConfigLog.Debugf("PLMN config changed %+v. Updating NF registration", newPlmnConfig)
		p.currentPlmnConfig = newPlmnConfig
		p.plmnConfigChan <- p.currentPlmnConfig
	}
	factory.NssfConfig.Configuration.SupportedNssaiInPlmnList = newSupportedNssai
}

func convertPlmnSnssaiList(newConfig []nfConfigApi.PlmnSnssai) ([]models.PlmnId, factory.SupportedNssaiInPlmn) {
	newPlmnList := make([]models.PlmnId, 0, len(newConfig))
	newSupportedNssais := make(factory.SupportedNssaiInPlmn)

	for _, plmnSnssai := range newConfig {
		newPlmn := models.PlmnId{
			Mcc: plmnSnssai.PlmnId.Mcc,
			Mnc: plmnSnssai.PlmnId.Mnc,
		}
		newPlmnList = append(newPlmnList, newPlmn)
		newSnssaiSet := make(map[models.Snssai]struct{})
		for _, snssai := range plmnSnssai.SNssaiList {
			newSnssai := models.Snssai{
				Sst: snssai.Sst,
			}
			if snssai.Sd != nil {
				newSnssai.Sd = *snssai.Sd
			}
			newSnssaiSet[newSnssai] = struct{}{}
		}

		newSupportedNssais[newPlmn] = newSnssaiSet
	}

	return newPlmnList, newSupportedNssais
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
