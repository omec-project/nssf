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
	"github.com/omec-project/openapi/models"
	utilLogger "github.com/omec-project/util/logger"
)

const (
	NSSF_EXPECTED_CONFIG_VERSION = "1.0.0"
)

type Config struct {
	Info          *Info              `yaml:"info"`
	Configuration *Configuration     `yaml:"configuration"`
	Logger        *utilLogger.Logger `yaml:"logger"`
	CfgLocation   string
	Subscriptions []Subscription `yaml:"subscriptions,omitempty"`
}

type Info struct {
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

const (
	NSSF_DEFAULT_IPV4     = "127.0.0.31"
	NSSF_DEFAULT_PORT_INT = 8000
)

type Configuration struct {
	NssfName                 string               `yaml:"nssfName,omitempty"`
	Sbi                      *Sbi                 `yaml:"sbi"`
	ServiceNameList          []models.ServiceName `yaml:"serviceNameList"`
	NrfUri                   string               `yaml:"nrfUri"`
	WebuiUri                 string               `yaml:"webuiUri"`
	SupportedNssaiInPlmnList SupportedNssaiInPlmn
	NsiList                  []NsiConfig             `yaml:"nsiList,omitempty"`
	AmfSetList               []AmfSetConfig          `yaml:"amfSetList"`
	AmfList                  []AmfConfig             `yaml:"amfList"`
	TaList                   []TaConfig              `yaml:"taList"`
	MappingListFromPlmn      []MappingFromPlmnConfig `yaml:"mappingListFromPlmn"`
}

type Sbi struct {
	Scheme models.UriScheme `yaml:"scheme"`
	TLS    *TLS             `yaml:"tls"`
	// Currently only support IPv4 and thus `Ipv4Addr` field shall not be empty
	RegisterIPv4 string `yaml:"registerIPv4,omitempty"` // IP that is registered at NRF.
	// IPv6Addr string `yaml:"ipv6Addr,omitempty"`
	BindingIPv4 string `yaml:"bindingIPv4,omitempty"` // IP used to run the server in the node.
	Port        int    `yaml:"port"`
}

type TLS struct {
	PEM string `yaml:"pem,omitempty"`
	Key string `yaml:"key,omitempty"`
}

type AmfConfig struct {
	NfId                           string                                  `yaml:"nfId"`
	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type TaConfig struct {
	Tai                  *models.Tai               `yaml:"tai"`
	AccessType           *models.AccessType        `yaml:"accessType"`
	SupportedSnssaiList  []models.Snssai           `yaml:"supportedSnssaiList"`
	RestrictedSnssaiList []models.RestrictedSnssai `yaml:"restrictedSnssaiList,omitempty"`
}

type SupportedNssaiInPlmn map[models.PlmnId]map[models.Snssai]struct{}

type NsiConfig struct {
	Snssai             *models.Snssai          `yaml:"snssai"`
	NsiInformationList []models.NsiInformation `yaml:"nsiInformationList"`
}

type AmfSetConfig struct {
	AmfSetId                       string                                  `yaml:"amfSetId"`
	AmfList                        []string                                `yaml:"amfList,omitempty"`
	NrfAmfSet                      string                                  `yaml:"nrfAmfSet,omitempty"`
	SupportedNssaiAvailabilityData []models.SupportedNssaiAvailabilityData `yaml:"supportedNssaiAvailabilityData"`
}

type MappingFromPlmnConfig struct {
	OperatorName    string                   `yaml:"operatorName,omitempty"`
	HomePlmnId      *models.PlmnId           `yaml:"homePlmnId"`
	MappingOfSnssai []models.MappingOfSnssai `yaml:"mappingOfSnssai"`
}

type Subscription struct {
	SubscriptionData *models.NssfEventSubscriptionCreateData `yaml:"subscriptionData"`
	SubscriptionId   string                                  `yaml:"subscriptionId"`
}

func (c *Config) GetVersion() string {
	if c.Info != nil && c.Info.Version != "" {
		return c.Info.Version
	}
	return ""
}
