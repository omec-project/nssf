// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

/*
 * NSSF Service
 */

package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	grpcClient "github.com/omec-project/config5g/proto/client"
	protos "github.com/omec-project/config5g/proto/sdcoreConfig"
	"github.com/omec-project/nssf/consumer"
	"github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/metrics"
	"github.com/omec-project/nssf/nssaiavailability"
	"github.com/omec-project/nssf/nsselection"
	"github.com/omec-project/nssf/util"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/http2_util"
	utilLogger "github.com/omec-project/util/logger"
	"github.com/omec-project/util/path_util"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type NSSF struct{}

type (
	// Config information.
	Config struct {
		nssfcfg string
	}
)

var config Config

var nssfCLi = []cli.Flag{
	cli.StringFlag{
		Name:  "free5gccfg",
		Usage: "common config file",
	},
	cli.StringFlag{
		Name:  "nssfcfg",
		Usage: "config file",
	},
}

var (
	KeepAliveTimer      *time.Timer
	KeepAliveTimerMutex sync.Mutex
)

func (*NSSF) GetCliCmd() (flags []cli.Flag) {
	return nssfCLi
}

func (nssf *NSSF) Initialize(c *cli.Context) error {
	config = Config{
		nssfcfg: c.String("nssfcfg"),
	}

	if config.nssfcfg != "" {
		if err := factory.InitConfigFactory(config.nssfcfg); err != nil {
			return err
		}
	} else {
		DefaultNssfConfigPath := path_util.Free5gcPath("free5gc/config/nssfcfg.yaml")
		if err := factory.InitConfigFactory(DefaultNssfConfigPath); err != nil {
			return err
		}
	}

	context.InitNssfContext()

	nssf.setLogLevel()

	if err := factory.CheckConfigVersion(); err != nil {
		return err
	}

	if os.Getenv("MANAGED_BY_CONFIG_POD") == "true" {
		logger.InitLog.Infoln("MANAGED_BY_CONFIG_POD is true")
		go manageGrpcClient(factory.NssfConfig.Configuration.WebuiUri)
	} else {
		go func() {
			logger.CfgLog.Infoln("use helm chart config")
			factory.ConfigPodTrigger <- true
		}()
	}
	return nil
}

// manageGrpcClient connects the config pod GRPC server and subscribes the config changes.
// Then it updates NSSF configuration.
func manageGrpcClient(webuiUri string) {
	var configChannel chan *protos.NetworkSliceResponse
	var client grpcClient.ConfClient
	var stream protos.ConfigService_NetworkSliceSubscribeClient
	var err error
	count := 0
	for {
		if client != nil {
			if client.CheckGrpcConnectivity() != "ready" {
				time.Sleep(time.Second * 30)
				count++
				if count > 5 {
					err = client.GetConfigClientConn().Close()
					if err != nil {
						logger.InitLog.Infof("failing ConfigClient is not closed properly: %+v", err)
					}
					client = nil
					count = 0
				}
				logger.InitLog.Infoln("checking the connectivity readiness")
				continue
			}

			if stream == nil {
				stream, err = client.SubscribeToConfigServer()
				if err != nil {
					logger.InitLog.Infof("failing SubscribeToConfigServer: %+v", err)
					continue
				}
			}

			if configChannel == nil {
				configChannel = client.PublishOnConfigChange(true, stream)
				logger.InitLog.Infoln("PublishOnConfigChange is triggered")
				go factory.NssfConfig.UpdateConfig(configChannel)
				logger.InitLog.Infoln("NSSF updateConfig is triggered")
			}
		} else {
			client, err = grpcClient.ConnectToConfigServer(webuiUri)
			stream = nil
			configChannel = nil
			logger.InitLog.Infoln("connecting to config server")
			if err != nil {
				logger.InitLog.Errorf("%+v", err)
			}
			continue
		}
	}
}

func (nssf *NSSF) setLogLevel() {
	if factory.NssfConfig.Logger == nil {
		logger.InitLog.Warnln("NSSF config without log level setting")
		return
	}

	if factory.NssfConfig.Logger.NSSF != nil {
		if factory.NssfConfig.Logger.NSSF.DebugLevel != "" {
			if level, err := zapcore.ParseLevel(factory.NssfConfig.Logger.NSSF.DebugLevel); err != nil {
				logger.InitLog.Warnf("NSSF Log level [%s] is invalid, set to [info] level",
					factory.NssfConfig.Logger.NSSF.DebugLevel)
				logger.SetLogLevel(zap.InfoLevel)
			} else {
				logger.InitLog.Infof("NSSF Log level is set to [%s] level", level)
				logger.SetLogLevel(level)
			}
		} else {
			logger.InitLog.Infoln("NSSF Log level not set. Default set to [info] level")
			logger.SetLogLevel(zap.InfoLevel)
		}
	}
}

func (nssf *NSSF) FilterCli(c *cli.Context) (args []string) {
	for _, flag := range nssf.GetCliCmd() {
		name := flag.GetName()
		value := fmt.Sprint(c.Generic(name))
		if value == "" {
			continue
		}

		args = append(args, "--"+name, value)
	}
	return args
}

func (nssf *NSSF) Start() {
	logger.InitLog.Infoln("server started")

	router := utilLogger.NewGinWithZap(logger.GinLog)

	nssaiavailability.AddService(router)
	nsselection.AddService(router)

	go metrics.InitMetrics()

	self := context.NSSF_Self()
	addr := fmt.Sprintf("%s:%d", self.BindingIPv4, self.SBIPort)

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		nssf.Terminate()
		os.Exit(0)
	}()

	go nssf.registerNF()

	server, err := http2_util.NewServer(addr, util.NSSF_LOG_PATH, router)

	if server == nil {
		logger.InitLog.Errorf("initialize HTTP server failed: %+v", err)
		return
	}

	if err != nil {
		logger.InitLog.Warnf("initialize HTTP server: +%v", err)
	}

	serverScheme := factory.NssfConfig.Configuration.Sbi.Scheme
	if serverScheme == "http" {
		err = server.ListenAndServe()
	} else if serverScheme == "https" {
		err = server.ListenAndServeTLS(self.PEM, self.Key)
	}

	if err != nil {
		logger.InitLog.Fatalf("HTTP server setup failed: %+v", err)
	}
}

func (nssf *NSSF) Exec(c *cli.Context) error {
	logger.InitLog.Debugln("args:", c.String("nssfcfg"))
	args := nssf.FilterCli(c)
	logger.InitLog.Debugln("filter:", args)
	command := exec.Command("./nssf", args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	wg := sync.WaitGroup{}
	goRoutines := 3
	wg.Add(goRoutines)
	go func() {
		in := bufio.NewScanner(stdout)
		for in.Scan() {
			logger.InitLog.Infoln(in.Text())
		}
		wg.Done()
	}()

	stderr, err := command.StderrPipe()
	if err != nil {
		logger.InitLog.Fatalln(err)
	}
	go func() {
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			logger.InitLog.Infoln(in.Text())
		}
		wg.Done()
	}()

	go func() {
		if err = command.Start(); err != nil {
			logger.InitLog.Errorf("NSSF start error: %v", err)
		}
		wg.Done()
	}()

	wg.Wait()

	return err
}

func (nssf *NSSF) Terminate() {
	logger.InitLog.Infoln("terminating NSSF...")
	// deregister with NRF
	problemDetails, err := consumer.SendDeregisterNFInstance()
	if problemDetails != nil {
		logger.InitLog.Errorf("deregister NF instance Failed Problem[%+v]", problemDetails)
	} else if err != nil {
		logger.InitLog.Errorf("deregister NF instance Error[%+v]", err)
	} else {
		logger.InitLog.Infoln("deregister from NRF successfully")
	}

	logger.InitLog.Infoln("NSSF terminated")
}

func (nssf *NSSF) StartKeepAliveTimer(nfProfile models.NfProfile) {
	KeepAliveTimerMutex.Lock()
	defer KeepAliveTimerMutex.Unlock()
	nssf.StopKeepAliveTimer()
	if nfProfile.HeartBeatTimer == 0 {
		nfProfile.HeartBeatTimer = 60
	}
	logger.InitLog.Infof("started KeepAlive Timer: %v sec", nfProfile.HeartBeatTimer)
	// AfterFunc starts timer and waits for KeepAliveTimer to elapse and then calls nssf.UpdateNF function
	KeepAliveTimer = time.AfterFunc(time.Duration(nfProfile.HeartBeatTimer)*time.Second, nssf.UpdateNF)
}

func (nssf *NSSF) StopKeepAliveTimer() {
	if KeepAliveTimer != nil {
		logger.InitLog.Infoln("stopped KeepAlive Timer")
		KeepAliveTimer.Stop()
		KeepAliveTimer = nil
	}
}

func (nssf *NSSF) BuildAndSendRegisterNFInstance() (models.NfProfile, error) {
	self := context.NSSF_Self()
	profile, err := consumer.BuildNFProfile(self)
	if err != nil {
		logger.InitLog.Errorf("build NSSF Profile Error: %v", err)
		return profile, err
	}
	logger.InitLog.Infof("NSSF Profile Registering to NRF: %v", profile)
	// Indefinite attempt to register until success
	profile, _, self.NfId, err = consumer.SendRegisterNFInstance(self.NrfUri, self.NfId, profile)
	return profile, err
}

// UpdateNF is the callback function, this is called when keepalivetimer elapsed
func (nssf *NSSF) UpdateNF() {
	KeepAliveTimerMutex.Lock()
	defer KeepAliveTimerMutex.Unlock()
	if KeepAliveTimer == nil {
		logger.InitLog.Warnln("keepAlive timer has been stopped")
		return
	}
	// setting default value 60 sec
	var heartBeatTimer int32 = 60
	pitem := models.PatchItem{
		Op:    "replace",
		Path:  "/nfStatus",
		Value: "REGISTERED",
	}
	var patchItem []models.PatchItem
	patchItem = append(patchItem, pitem)
	nfProfile, problemDetails, err := consumer.SendUpdateNFInstance(patchItem)
	if problemDetails != nil {
		logger.InitLog.Errorf("NSSF update to NRF ProblemDetails[%v]", problemDetails)
		// 5xx response from NRF, 404 Not Found, 400 Bad Request
		if (problemDetails.Status >= 500 && problemDetails.Status <= 599) ||
			problemDetails.Status == 404 || problemDetails.Status == 400 {
			// register with NRF full profile
			nfProfile, err = nssf.BuildAndSendRegisterNFInstance()
			if err != nil {
				logger.InitLog.Errorf("NSSF update to NRF Error[%s]", err.Error())
			}
		}
	} else if err != nil {
		logger.InitLog.Errorf("NSSF update to NRF Error[%s]", err.Error())
		nfProfile, err = nssf.BuildAndSendRegisterNFInstance()
		if err != nil {
			logger.InitLog.Errorf("NSSF update to NRF Error[%s]", err.Error())
		}
	}

	if nfProfile.HeartBeatTimer != 0 {
		// use hearbeattimer value with received timer value from NRF
		heartBeatTimer = nfProfile.HeartBeatTimer
	}
	logger.InitLog.Debugf("restarted KeepAlive Timer: %v sec", heartBeatTimer)
	// restart timer with received HeartBeatTimer value
	KeepAliveTimer = time.AfterFunc(time.Duration(heartBeatTimer)*time.Second, nssf.UpdateNF)
}

func (nssf *NSSF) registerNF() {
	for msg := range factory.ConfigPodTrigger {
		logger.InitLog.Infof("minimum configuration from config pod available %v", msg)
		self := context.NSSF_Self()
		profile, err := consumer.BuildNFProfile(self)
		if err != nil {
			logger.InitLog.Errorln("build profile failed")
		}

		var newNrfUri string
		var prof models.NfProfile
		// send registration with updated PLMN Ids.
		prof, newNrfUri, self.NfId, err = consumer.SendRegisterNFInstance(self.NrfUri, profile.NfInstanceId, profile)
		if err == nil {
			nssf.StartKeepAliveTimer(prof)
			logger.CfgLog.Infoln("sent register NFInstance with updated profile")
			self.NrfUri = newNrfUri
		} else {
			logger.CfgLog.Errorf("send register NFInstance Error[%s]", err.Error())
		}
	}
}
