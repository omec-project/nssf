// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"sync"

	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/util/nfconfig"
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
	NssfConfig = Config{}

	if err := nfconfig.LoadYAML(f, &NssfConfig); err != nil {
		return err
	}
	webuiUri, err := nfconfig.ResolveWebuiUri(NssfConfig.Configuration.WebuiUri, logger.CfgLog)
	if err != nil {
		return err
	}
	NssfConfig.Configuration.WebuiUri = webuiUri
	return nil
}

func CheckConfigVersion() error {
	return nfconfig.CheckVersion(NssfConfig.GetVersion(), NSSF_EXPECTED_CONFIG_VERSION, logger.CfgLog)
}
