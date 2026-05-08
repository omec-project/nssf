// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NF Context for NSSF
 *
 * Configuration of NSSF itself shall be accessed with NSSF context
 * Configuration of network slices shall be accessed with configuration factory
 */

package context

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/openapi/v2"
	"github.com/omec-project/openapi/v2/models"
)

var nssfContext = NSSFContext{}

const port int = 29510

// Initialize NSSF context with default value
func init() {
	nssfContext.NfId = uuid.New().String()

	nssfContext.Name = "NSSF"

	nssfContext.UriScheme = models.URISCHEME_HTTPS
	nssfContext.RegisterIPv4 = factory.NSSF_DEFAULT_IPV4
	nssfContext.SBIPort = factory.NSSF_DEFAULT_PORT_INT

	serviceName := []models.ServiceName{
		models.SERVICENAME_NNSSF_NSSELECTION,
		models.SERVICENAME_NNSSF_NSSAIAVAILABILITY,
	}
	nssfContext.NfService = initNfService(serviceName, "1.0.0")

	nssfContext.NrfUri = fmt.Sprintf("%s://%s:%d", models.URISCHEME_HTTPS, nssfContext.RegisterIPv4, port)
}

type NSSFContext struct {
	NfId         string
	Name         string
	UriScheme    models.UriScheme
	RegisterIPv4 string
	BindingIPv4  string
	Key          string
	PEM          string
	NfService    map[models.ServiceName]models.NFService
	NrfUri       string
	SBIPort      int
}

// Initialize NSSF context with configuration factory
func InitNssfContext() {
	if !factory.Configured {
		logger.ContextLog.Warnf("NSSF is not configured")
		return
	}
	nssfConfig := factory.NssfConfig

	if nssfConfig.Configuration.NssfName != "" {
		nssfContext.Name = nssfConfig.Configuration.NssfName
	}

	nssfContext.UriScheme = nssfConfig.Configuration.Sbi.Scheme
	nssfContext.RegisterIPv4 = nssfConfig.Configuration.Sbi.RegisterIPv4
	nssfContext.SBIPort = nssfConfig.Configuration.Sbi.Port
	nssfContext.BindingIPv4 = os.Getenv(nssfConfig.Configuration.Sbi.BindingIPv4)
	if tls := nssfConfig.Configuration.Sbi.TLS; tls != nil {
		if tls.Key != "" {
			nssfContext.Key = tls.Key
		}
		if tls.PEM != "" {
			nssfContext.PEM = tls.PEM
		}
	}
	if nssfContext.BindingIPv4 != "" {
		logger.ContextLog.Infoln("parsing ServerIPv4 address from ENV Variable")
	} else {
		nssfContext.BindingIPv4 = nssfConfig.Configuration.Sbi.BindingIPv4
		if nssfContext.BindingIPv4 == "" {
			logger.ContextLog.Warnln("error parsing ServerIPv4 address as string. Using the 0.0.0.0 address as default")
			nssfContext.BindingIPv4 = "0.0.0.0"
		}
	}

	// NF service API versions must track the served SBI routes, not the config schema version.
	nssfContext.NfService = initNfService(nssfConfig.Configuration.ServiceNameList, factory.NSSF_EXPECTED_CONFIG_VERSION)

	if nssfConfig.Configuration.NrfUri != "" {
		nssfContext.NrfUri = nssfConfig.Configuration.NrfUri
	} else {
		logger.InitLog.Warnln("NRF Uri is empty. Using localhost as NRF IPv4 address")
		nssfContext.NrfUri = fmt.Sprintf("%s://%s:%d", nssfContext.UriScheme, "127.0.0.1", port)
	}
}

func initNfService(serviceName []models.ServiceName, version string) (
	nfService map[models.ServiceName]models.NFService,
) {
	versionUri := "v" + strings.Split(version, ".")[0]
	nfService = make(map[models.ServiceName]models.NFService)
	for idx, name := range serviceName {
		ipEndPoint := models.NewIpEndPoint()
		ipEndPoint.SetIpv4Address(nssfContext.RegisterIPv4)
		ipEndPoint.SetTransport(models.TRANSPORTPROTOCOL_TCP)
		ipEndPoint.SetPort(int32(nssfContext.SBIPort))
		nfService[name] = models.NFService{
			ServiceInstanceId: strconv.Itoa(idx),
			ServiceName:       name,
			Versions: []models.NFServiceVersion{
				{
					ApiFullVersion:  version,
					ApiVersionInUri: versionUri,
				},
			},
			Scheme:          nssfContext.UriScheme,
			NfServiceStatus: models.NFSERVICESTATUS_REGISTERED,
			ApiPrefix:       openapi.PtrString(GetIpv4Uri()),
			IpEndPoints:     []models.IpEndPoint{*ipEndPoint},
		}
	}

	return
}

func GetIpv4Uri() string {
	return fmt.Sprintf("%s://%s:%d", nssfContext.UriScheme, nssfContext.RegisterIPv4, nssfContext.SBIPort)
}

func NSSF_Self() *NSSFContext {
	return &nssfContext
}
