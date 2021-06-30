// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

/*
 * NSSF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/free5gc/nssf/logger"
	gClient "github.com/omec-project/nssf/proto/client"
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
	if content, err := ioutil.ReadFile(f); err != nil {
		return err
	} else {
		NssfConfig = Config{}

		if yamlErr := yaml.Unmarshal(content, &NssfConfig); yamlErr != nil {
			return yamlErr
		}

		Configured = true
		commChannel := gClient.ConfigWatcher()
		go NssfConfig.updateConfig(commChannel)

	}

	return nil
}

func CheckConfigVersion() error {
	currentVersion := NssfConfig.GetVersion()

	if currentVersion != NSSF_EXPECTED_CONFIG_VERSION {
		return fmt.Errorf("config version is [%s], but expected is [%s].",
			currentVersion, NSSF_EXPECTED_CONFIG_VERSION)
	}

	logger.CfgLog.Infof("config version [%s]", currentVersion)

	return nil
}
