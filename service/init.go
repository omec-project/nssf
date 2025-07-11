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
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	nssfContext "github.com/omec-project/nssf/context"
	"github.com/omec-project/nssf/factory"
	"github.com/omec-project/nssf/logger"
	"github.com/omec-project/nssf/metrics"
	"github.com/omec-project/nssf/nfregistration"
	"github.com/omec-project/nssf/nssaiavailability"
	"github.com/omec-project/nssf/nsselection"
	"github.com/omec-project/nssf/polling"
	"github.com/omec-project/openapi/models"
	"github.com/omec-project/util/http2_util"
	utilLogger "github.com/omec-project/util/logger"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type NSSF struct{}

type (
	// Config information.
	Config struct {
		cfg string
	}
)

var config Config

var nssfCLi = []cli.Flag{
	&cli.StringFlag{
		Name:     "cfg",
		Usage:    "nssf config file",
		Required: true,
	},
}

func (*NSSF) GetCliCmd() (flags []cli.Flag) {
	return nssfCLi
}

func (nssf *NSSF) Initialize(c *cli.Command) error {
	config = Config{
		cfg: c.String("cfg"),
	}

	absPath, err := filepath.Abs(config.cfg)
	if err != nil {
		logger.CfgLog.Errorln(err)
		return err
	}

	if err := factory.InitConfigFactory(absPath); err != nil {
		return err
	}

	nssf.setLogLevel()

	if err := factory.CheckConfigVersion(); err != nil {
		return err
	}

	factory.NssfConfig.CfgLocation = absPath

	factory.Configured = true
	nssfContext.InitNssfContext()
	return nil
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

func (nssf *NSSF) FilterCli(c *cli.Command) (args []string) {
	for _, flag := range nssf.GetCliCmd() {
		name := flag.Names()[0]
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

	self := nssfContext.NSSF_Self()
	addr := fmt.Sprintf("%s:%d", self.BindingIPv4, self.SBIPort)

	plmnConfigChan := make(chan []models.PlmnId, 1)
	ctx, cancelServices := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		polling.StartPollingService(ctx, factory.NssfConfig.Configuration.WebuiUri, plmnConfigChan)
	}()
	go func() {
		defer wg.Done()
		nfregistration.StartNfRegistrationService(ctx, plmnConfigChan)
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel
		nssf.Terminate(cancelServices, &wg)
		os.Exit(0)
	}()

	sslLog := filepath.Dir(factory.NssfConfig.CfgLocation) + "/sslkey.log"
	server, err := http2_util.NewServer(addr, sslLog, router)

	if server == nil {
		logger.InitLog.Errorf("initialize HTTP server failed: %+v", err)
		return
	}

	if err != nil {
		logger.InitLog.Warnf("initialize HTTP server: +%v", err)
	}

	serverScheme := factory.NssfConfig.Configuration.Sbi.Scheme
	switch serverScheme {
	case "http":
		err = server.ListenAndServe()
	case "https":
		err = server.ListenAndServeTLS(self.PEM, self.Key)
	default:
		logger.InitLog.Fatalf("HTTP server setup failed: invalid server scheme %+v", serverScheme)
		return
	}

	if err != nil {
		logger.InitLog.Fatalf("HTTP server setup failed: %+v", err)
	}
}

func (nssf *NSSF) Exec(c *cli.Command) error {
	logger.InitLog.Debugln("args:", c.String("cfg"))
	args := nssf.FilterCli(c)
	logger.InitLog.Debugln("filter:", args)
	command := exec.Command("nssf", args...)

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

func (nssf *NSSF) Terminate(cancelServices context.CancelFunc, wg *sync.WaitGroup) {
	logger.InitLog.Infoln("terminating NSSF")
	cancelServices()
	nfregistration.DeregisterNF()
	wg.Wait()
	logger.InitLog.Infoln("NSSF terminated")
}
