package server

import (
	context "context"
	protos "github.com/badhrinathpa/nssf/proto/omec5gconfig"
	"google.golang.org/grpc"
	"log"
	"net"
)

type PlmnId struct {
	MCC string
	MNC string
}

type SupportedPlmnList struct {
	PlmnIdList []PlmnId
}

type Nssai struct {
	sst string
	sd  string
}

type SupportedNssaiList struct {
	NssaiList []Nssai
}

type SupportedNssaiInPlmnList struct {
	Plmn       PlmnId
	SnssaiList SupportedNssaiList
}

type ServerConfig struct {
	suppNssaiPlmnList SupportedNssaiInPlmnList
	SuppPlmnList      SupportedPlmnList
}

type ConfigServer struct {
	protos.ConfigServiceServer
	serverCfg ServerConfig
	Version   uint32
}

func StartServer(host string, confServ *ConfigServer) {
	log.Println("start config server")
	lis, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	protos.RegisterConfigServiceServer(grpcServer, confServ)
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (c *ConfigServer) Read(ctx context.Context, rReq *protos.ReadRequest) (*protos.ReadResponse, error) {
	log.Println("Handle Read config request")
	rResp := &protos.ReadResponse{}
	rCfg := &protos.Config{}
	rResp.ReadConfig = rCfg
	suppPlmnList := &protos.SupportedPlmnList{}
	for _, pl := range c.serverCfg.SuppPlmnList.PlmnIdList {
		log.Println("mcc: ", pl.MCC)
		log.Println("mnc: ", pl.MNC)
		plmnId := &protos.PlmnId{
			Mcc: pl.MCC,
			Mnc: pl.MNC,
		}
		suppPlmnList.PlmnIds = append(suppPlmnList.PlmnIds, plmnId)
	}
	rCfg.SuppPlmnList = suppPlmnList
	return rResp, nil
}

func (c *ConfigServer) Write(ctx context.Context, wReq *protos.WriteRequest) (*protos.WriteResponse, error) {
	log.Println("Handle write request")
	wResp := &protos.WriteResponse{}
	wResp.WriteStatus = protos.Status_SUCCESS

	wCfg := wReq.WriteConfig
	suppPlmnList := wCfg.SuppPlmnList
	for _, pl := range suppPlmnList.PlmnIds {
		log.Println("mcc: ", pl.Mcc)
		log.Println("mnc: ", pl.Mnc)
		plmnId := PlmnId{
			MCC: pl.GetMcc(),
			MNC: pl.GetMnc(),
		}
		c.serverCfg.SuppPlmnList.PlmnIdList = append(c.serverCfg.SuppPlmnList.PlmnIdList, plmnId)
	}
	return wResp, nil
}
