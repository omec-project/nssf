// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package client

import (
	context "context"
	"github.com/omec-project/nssf/logger"
	protos "github.com/omec-project/nssf/proto/sdcoreConfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"math/rand"
	"os"
	"time"
)

var selfRestartCounter uint32
var configPodRestartCounter uint32 = 0

func init() {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	selfRestartCounter = r1.Uint32()
}

type PlmnId struct {
	MCC string
	MNC string
}

type Nssai struct {
	sst string
	sd  string
}

type ConfigClient struct {
	Client  protos.ConfigServiceClient
	Conn    *grpc.ClientConn
	Version string
}

func ConfigWatcher() chan *protos.NetworkSliceResponse {
	//var confClient *gClient.ConfigClient
    //TODO: use port from configmap. 
	confClient, err := CreateChannel("webui:9876", 10000)
	if err != nil {
		logger.GrpcLog.Errorf("create grpc channel to config pod failed. : ", err)
		return nil
	}
	commChan := make(chan *protos.NetworkSliceResponse)
	go readConfigInLoop(confClient, commChan)
	return commChan
}

func CreateChannel(host string, timeout uint32) (*ConfigClient, error) {
	logger.GrpcLog.Infoln("create config client")
	// Second, check to see if we can reuse the gRPC connection for a new P4RT client
	conn, err := GetConnection(host)
	if err != nil {
		logger.GrpcLog.Errorf("grpc connection failed")
		return nil, err
	}

	client := &ConfigClient{
		Client: protos.NewConfigServiceClient(conn),
		Conn:   conn,
	}

	return client, nil
}

var kacp = keepalive.ClientParameters{
	Time:                20 * time.Second, // send pings every 20 seconds if there is no activity
	Timeout:             2 * time.Second,  // wait 1 second for ping ack before considering the connection dead
	PermitWithoutStream: true,             // send pings even without active streams
}

var retryPolicy = `{
		"methodConfig": [{
		  "name": [{"service": "grpc.Config"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".01s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 1.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }}]}`

func GetConnection(host string) (conn *grpc.ClientConn, err error) {
	/* get connection */
	logger.GrpcLog.Infoln("Dial grpc connection - ",host)
	conn, err = grpc.Dial(host, grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp), grpc.WithDefaultServiceConfig(retryPolicy))
	if err != nil {
		logger.GrpcLog.Errorln("grpc dial err: ", err)
		return nil, err
	}
	//defer conn.Close()
	return conn, err
}

func readConfigInLoop(confClient *ConfigClient, commChan chan *protos.NetworkSliceResponse) {
	myid := os.Getenv("HOSTNAME")
	configReadTimeout := time.NewTicker(5000 * time.Millisecond)
	for {
		select {
		case <-configReadTimeout.C:
			status := confClient.Conn.GetState()
			if status == connectivity.Ready {
				rreq := &protos.NetworkSliceRequest{RestartCounter: selfRestartCounter, ClientId: myid}
				rsp, err := confClient.Client.GetNetworkSlice(context.Background(), rreq)
				if err != nil {
					logger.GrpcLog.Errorln("read Network Slice config from webconsole failed : ", err)
					continue
				}
				logger.GrpcLog.Debugf("#Network Slices %v, RC of configpod %v ", len(rsp.NetworkSlice), rsp.RestartCounter)
				if configPodRestartCounter == 0 || (configPodRestartCounter == rsp.RestartCounter) {
					// first time connection or config update
					configPodRestartCounter = rsp.RestartCounter
					if len(rsp.NetworkSlice) > 0 {
						// always carries full config copy
						logger.GrpcLog.Infoln("First time config Received ", rsp)
						commChan <- rsp
					} else if rsp.ConfigUpdated == 1 {
						// config delete , all slices deleted
						logger.GrpcLog.Infoln("Complete config deleted ")
						commChan <- rsp
					}
				} else if len(rsp.NetworkSlice) > 0 {
					logger.GrpcLog.Errorf("Config received after config Pod restart")
					//config received after config pod restart
					configPodRestartCounter = rsp.RestartCounter
					commChan <- rsp
				} else {
					logger.GrpcLog.Errorf("Config Pod is restarted and no config received")
				}
			} else {
				logger.GrpcLog.Errorln("read Network Slice config from webconsole skipped. GRPC channel down ")
			}
		}
	}
}

func (c *ConfigClient) ReadNetworkSliceConfig() error {
	logger.GrpcLog.Infoln("ReadNetworkSliceConfig  request")
	myid := os.Getenv("HOSTNAME")
	rreq := &protos.NetworkSliceRequest{RestartCounter: selfRestartCounter, ClientId: myid}
	rsp, err := c.Client.GetNetworkSlice(context.Background(), rreq)
	if err != nil {
		logger.GrpcLog.Errorf("GRPC error - GetNetworkSlice %v", err)
		return err
	}
	logger.GrpcLog.Infof("Number of Network Slices available %v, restart counter  of configpod %v ", len(rsp.NetworkSlice), rsp.RestartCounter)
    if configPodRestartCounter == 0 || (configPodRestartCounter == rsp.RestartCounter) {
      // first time connection 
      configPodRestartCounter = rsp.RestartCounter
	  for n := 0; n < len(rsp.NetworkSlice); n++ {
		ns := rsp.NetworkSlice[n]
		logger.GrpcLog.Infoln("Network Slice Name ", ns.Name)
        if ns.Site != nil {
		    logger.GrpcLog.Infoln("Network Slice has site name present ")
            site := ns.Site
		    logger.GrpcLog.Infoln("Site name ", site.SiteName)
            if site.Plmn != nil {
                plmn := site.Plmn
		        logger.GrpcLog.Infoln("Plmn mcc ", plmn.Mcc)
            } else {
		        logger.GrpcLog.Infoln("Plmn not present in the message ")
            }
        }
	  }
    } else if len(rsp.NetworkSlice) > 0 {
      // Config copy Received after Config Pod has restarted
      configPodRestartCounter = rsp.RestartCounter
	  for n := 0; n < len(rsp.NetworkSlice); n++ {
		ns := rsp.NetworkSlice[n]
		logger.GrpcLog.Infoln("Network Slice Name ", ns.Name)
	  }
    } else {
		logger.GrpcLog.Errorf("Config Pod is restarted and not config received")
    }
	return nil
}
