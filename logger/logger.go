// Copyright (C) 2024-Present Intel Corporation
// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
// Copyright 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	utilLogger "github.com/omec-project/util/logger"
	"go.uber.org/zap/zapcore"
)

var component = utilLogger.NewComponent("NSSF")

var (
	AppLog             = component.Category("App")
	ContextLog         = component.Category("CTX")
	FactoryLog         = component.Category("Factory")
	HandlerLog         = component.Category("HDLR")
	InitLog            = component.Category("Init")
	CfgLog             = component.Category("CFG")
	Nsselection        = component.Category("NsSelect")
	Nssaiavailability  = component.Category("NssaiAvail")
	Util               = component.Category("Util")
	ConsumerLog        = component.Category("Consumer")
	GinLog             = component.Category("GIN")
	PollConfigLog      = component.Category("PollConfig")
	NrfRegistrationLog = component.Category("NrfRegistration")
)

// SetLogLevel: set the log level (panic|fatal|error|warn|info|debug)
func SetLogLevel(level zapcore.Level) {
	InitLog.Infoln("set log level:", level)
	component.SetLevel(level)
}
