// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Testing Utility
 */

package test

import (
	"github.com/omec-project/nssf/producer"
)

var (
	ConfigFileFromArgs string
	DefaultConfigFile  string = "conf/test_nssf_config.yaml"
)

type TestingUtil struct {
	ConfigFile string
}

type TestingNsselection struct {
	GenerateNonRoamingQueryParameter func() producer.NsselectionQueryParameter
	GenerateRoamingQueryParameter    func() producer.NsselectionQueryParameter
	ConfigFile                       string
}

type TestingNssaiavailability struct {
	ConfigFile string

	NfId string

	SubscriptionId string

	NfNssaiAvailabilityUri string
}
