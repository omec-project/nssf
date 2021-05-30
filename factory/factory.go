/*
 * NSSF Configuration Factory
 */

package factory

import (
	"fmt"
	"io/ioutil"
	"sync"
	"reflect"

	"gopkg.in/yaml.v2"

	"github.com/free5gc/nssf/logger"
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
	}

	return nil
}

func UpdateAmfConfig(f string) error {
	if content, err := ioutil.ReadFile(f); err != nil {
		return err
	} else {
		var nssfConfig Config

		if yamlErr := yaml.Unmarshal(content, &nssfConfig); yamlErr != nil {
			return yamlErr
		}

		if reflect.DeepEqual(NssfConfig.Configuration.NssfName, nssfConfig.Configuration.NssfName) == false {
			logger.CfgLog.Infoln("updated NSSF Name is changed to ", nssfConfig.Configuration.NssfName)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.Sbi, nssfConfig.Configuration.Sbi) == false {
			logger.CfgLog.Infoln("updated Sbi is changed to ", nssfConfig.Configuration.NssfName)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.ServiceNameList, nssfConfig.Configuration.ServiceNameList) == false {
			logger.CfgLog.Infoln("updated ServiceNameList is changed to ", nssfConfig.Configuration.ServiceNameList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.NrfUri, nssfConfig.Configuration.NrfUri) == false {
			logger.CfgLog.Infoln("updated NrfUri is changed to ", nssfConfig.Configuration.NrfUri)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.SupportedPlmnList, nssfConfig.Configuration.SupportedPlmnList) == false {
			logger.CfgLog.Infoln("updated SupportedPlmnList is changed to ", nssfConfig.Configuration.SupportedPlmnList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.SupportedNssaiInPlmnList, nssfConfig.Configuration.SupportedNssaiInPlmnList) == false {
			logger.CfgLog.Infoln("updated SupportedNssaiInPlmnList is changed to ", nssfConfig.Configuration.SupportedNssaiInPlmnList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.NsiList, nssfConfig.Configuration.NsiList) == false {
			logger.CfgLog.Infoln("updated NsiList is changed to ", nssfConfig.Configuration.NsiList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.AmfSetList, nssfConfig.Configuration.AmfSetList) == false {
			logger.CfgLog.Infoln("updated AmfSetList is changed to ", nssfConfig.Configuration.AmfSetList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.AmfList, nssfConfig.Configuration.AmfList) == false {
			logger.CfgLog.Infoln("updated AmfList is changed to ", nssfConfig.Configuration.AmfList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.TaList, nssfConfig.Configuration.TaList) == false {
			logger.CfgLog.Infoln("updated TaList is changed to ", nssfConfig.Configuration.TaList)
		}
		if reflect.DeepEqual(NssfConfig.Configuration.MappingListFromPlmn, nssfConfig.Configuration.MappingListFromPlmn) == false {
			logger.CfgLog.Infoln("updated MappingListFromPlmn is changed to ", nssfConfig.Configuration.MappingListFromPlmn)
		}
		NssfConfig = nssfConfig
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
