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
	"os"
	"sync"

	"gopkg.in/yaml.v2"

	"github.com/free5gc/nssf/logger"
	"github.com/omec-project/config5g/proto/client"
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

		roc := os.Getenv("MANAGED_BY_CONFIG_POD")
		if roc == "true" {
			logger.CfgLog.Infoln("MANAGED_BY_CONFIG_POD is true")
			commChannel := client.ConfigWatcher()
			go NssfConfig.updateConfig(commChannel)
		} else {
			go func() {
				logger.CfgLog.Infoln("Use helm chart config ")
				ConfigPodTrigger <- true
			}()
		}
		Configured = true
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
