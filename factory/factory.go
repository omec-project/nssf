// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Configuration Factory
 */

package factory

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/omec-project/nssf/logger"
	"gopkg.in/yaml.v2"
)

var (
	NssfConfig Config
	Configured bool
	ConfigLock sync.RWMutex
)

func init() {
	Configured = false
}

// TODO: Support configuration update from REST api
func InitConfigFactory(f string) error {
	content, err := os.ReadFile(f)
	if err != nil {
		return err
	}
	NssfConfig = Config{}

	if err = yaml.Unmarshal(content, &NssfConfig); err != nil {
		return err
	}

	if NssfConfig.Configuration.WebuiUri == "" {
		NssfConfig.Configuration.WebuiUri = "http://webui:5001"
		logger.CfgLog.Infof("webuiUri not set in configuration file. Using %v", NssfConfig.Configuration.WebuiUri)
		return nil
	}
	err = validateWebuiUri(NssfConfig.Configuration.WebuiUri)
	return err
}

func CheckConfigVersion() error {
	currentVersion := NssfConfig.GetVersion()

	if currentVersion != NSSF_EXPECTED_CONFIG_VERSION {
		return fmt.Errorf("config version is [%s], but expected is [%s]",
			currentVersion, NSSF_EXPECTED_CONFIG_VERSION)
	}

	logger.CfgLog.Infof("config version [%s]", currentVersion)

	return nil
}

func validateWebuiUri(uri string) error {
	parsedUrl, err := url.ParseRequestURI(uri)
	if err != nil {
		return err
	}
	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return fmt.Errorf("unsupported scheme for webuiUri: %s", parsedUrl.Scheme)
	}
	if parsedUrl.Hostname() == "" {
		return fmt.Errorf("missing host in webuiUri")
	}
	return nil
}
