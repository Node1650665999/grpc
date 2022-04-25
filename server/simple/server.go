package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"log"
	"net"
	"runtime/debug"
	"time"
)

//SearchService 定义服务，需实现了SearchServiceServer，这样该服务才能注册
type SearchService struct{}

//Search 实现了SearchServiceServer接口定义的方法,实现该方法就能填充我们的业务逻辑
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{Response: r.GetRequest() + " Server"}, nil
}

const PORT = "9001"

func main() {

	c    := GetTLSCredentials()
	opts := []grpc.ServerOption{
		//证书
		grpc.Creds(c),
		//注册中间件
		grpc_middleware.WithUnaryServerChain(
			RecoveryInterceptor,
			LoggingInterceptor,
		),
	}

	server := grpc.NewServer(opts...)
	pb.RegisterSearchServiceServer(server, &SearchService{})

	lis, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	server.Serve(lis)
}

func GetTLSCredentials() credentials.TransportCredentials {
	cert, err := tls.LoadX509KeyPair("../../cert/server.pem", "../../cert/server.key")
	if err != nil {
		log.Fatalf("tls.LoadX509KeyPair err: %v", err)
	}

	certPool := x509.NewCertPool()
	ca, err  := ioutil.ReadFile("../../cert/ca.pem")
	if err != nil {
		log.Fatalf("ioutil.ReadFile err: %v", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Fatalf("certPool.AppendCertsFromPEM err")
	}

	c := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})

	return c
}


type ErrorData struct {
	Code int
	Msg  string
}

func ErrorDataString(code int, msg string) string  {
	str,_ := json.Marshal(ErrorData{code, msg})
	return string(str)
}

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	//模拟超时
	//time.Sleep(10 * time.Second)

	//模拟错误
	/*if req!= "gRPC" {
		return nil, status.Errorf(codes.Internal, "cannot access!!!")
	}*/

	//模拟异常,将触发RecoveryInterceptor中间件
	//panic("some thing error")

	//附带自定义业务错误
	/*if req!= "gRPC" {
		selfError := ErrorDataString(3000, "自定义业务参数")
		sts,_ := status.New(codes.Internal, "Rpc self define err").WithDetails(proto.MessageV1(selfError))
		return nil, sts.Err()
	}*/


	//前置操作
	log.Printf("gRPC method: %s, %v", info.FullMethod, req)
	resp, err := handler(ctx, req)
	//后置操作
	log.Printf("gRPC method: %s, %v", info.FullMethod, resp)
	return resp, err
}

//AccessLog 定义了访问日志中间件
func AccessLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	requestLog := "access request log: method: %s, begin_time: %d, request: %v"
	beginTime := time.Now().Local().Unix()
	log.Printf(requestLog, info.FullMethod, beginTime, req)

	resp, err := handler(ctx, req)

	responseLog := "access response log: method: %s, begin_time: %d, end_time: %d, response: %v"
	endTime := time.Now().Local().Unix()
	log.Printf(responseLog, info.FullMethod, beginTime, endTime, resp)
	return resp, err
}

//ErrorLog 定义错误日志中间件
func ErrorLog(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		errLog := "error log: method: %s, code: %v, message: %v, details: %v"
		statusErr, _ := status.FromError(err)
		log.Printf(errLog, info.FullMethod, statusErr.Code(), statusErr.Err().Error(), statusErr.Details())
	}
	return resp, err
}

//RecoveryInterceptor 定义了异常捕获中间件,假使没有异常捕获,则服务无法提供响应,也就是说系统崩溃了
func RecoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			recoveryLog := "recovery log: method: %s, message: %v, stack: %s"
			log.Printf(recoveryLog, info.FullMethod, e, string(debug.Stack()))
		}
	}()

	return handler(ctx, req)
}
