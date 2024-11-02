// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Testing Utility
 */

package test

import (
	"flag"

	"github.com/omec-project/nssf/plugin"
)

var (
	ConfigFileFromArgs string
	DefaultConfigFile  string = "conf/test_nssf_config.yaml"
)

type TestingUtil struct {
	ConfigFile string
}

type TestingNsselection struct {
	GenerateNonRoamingQueryParameter func() plugin.NsselectionQueryParameter
	GenerateRoamingQueryParameter    func() plugin.NsselectionQueryParameter
	ConfigFile                       string
}

type TestingNssaiavailability struct {
	ConfigFile string

	NfId string

	SubscriptionId string

	NfNssaiAvailabilityUri string
}

func init() {
	flag.StringVar(&ConfigFileFromArgs, "config-file", DefaultConfigFile, "Configuration file")
	flag.Parse()
}
