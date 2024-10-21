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
	"os"
	"sync"
	"time"

	grpcClient "github.com/omec-project/config5g/proto/client"
	protos "github.com/omec-project/config5g/proto/sdcoreConfig"
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

// InitConfigFactory gets the NssfConfig and subscribes the config pod.
// This observes the GRPC client availability and connection status in a loop.
// When the GRPC server pod is restarted, GRPC connection status stuck in idle.
// If GRPC client does not exist, creates it. If client exists but GRPC connectivity is not ready,
// then it closes the existing client start a new client.
func InitConfigFactory(f string) error {
	if content, err := os.ReadFile(f); err != nil {
		return err
	} else {
		NssfConfig = Config{}

		if yamlErr := yaml.Unmarshal(content, &NssfConfig); yamlErr != nil {
			return yamlErr
		}
		if NssfConfig.Configuration.WebuiUri == "" {
			NssfConfig.Configuration.WebuiUri = "webui:9876"
		}
		if os.Getenv("MANAGED_BY_CONFIG_POD") == "true" {
			logger.CfgLog.Infoln("MANAGED_BY_CONFIG_POD is true")
			client, err := grpcClient.ConnectToConfigServer(NssfConfig.Configuration.WebuiUri)
			if err != nil {
				go updateConfig(client)
			}
			return err
		} else {
			go func() {
				logger.CfgLog.Infoln("Use helm chart config")
				ConfigPodTrigger <- true
			}()
		}
		Configured = true
	}

	return nil
}

// updateConfig connects the config pod GRPC server and subscribes the config changes
// then updates NSSF configuration
func updateConfig(client grpcClient.ConfClient) {
	var stream protos.ConfigService_NetworkSliceSubscribeClient
	var err error
	var configChannel chan *protos.NetworkSliceResponse
	for {
		if client != nil {
			stream, err = client.CheckGrpcConnectivity()
			if err != nil {
				logger.CfgLog.Errorf("%v", err)
				if stream != nil {
					time.Sleep(time.Second * 30)
					continue
				} else {
					err = client.GetConfigClientConn().Close()
					if err != nil {
						logger.CfgLog.Debugf("failing ConfigClient is not closed properly: %+v", err)
					}
					client = nil
					continue
				}
			}
			if configChannel == nil {
				configChannel = client.PublishOnConfigChange(true, stream)
				go NssfConfig.updateConfig(configChannel)
			}
		} else {
			client, err = grpcClient.ConnectToConfigServer(NssfConfig.Configuration.WebuiUri)
			if err != nil {
				logger.CfgLog.Errorf("%+v", err)
			}
			continue
		}
	}
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
