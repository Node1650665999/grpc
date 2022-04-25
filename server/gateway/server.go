package main

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "go-grpc/proto"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net/http"
	"strings"
)

//SearchService 定义服务，需实现了SearchServiceServer，这样该服务才能注册
type SearchService struct{}

//Search 实现了SearchServiceServer接口定义的方法,实现该方法就能填充我们的业务逻辑
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	return &pb.SearchResponse{Response: r.GetRequest() + " Server"}, nil
}

const PORT = "9001"
const ENDPOINT = ":" + PORT

func main() {
	httpServer := httpServer()
	grpcServer := grpcServer()
	gateway    := grpcGateway()

	httpServer.Handle("/", gateway)

	http.ListenAndServe(ENDPOINT, grpcHandlerFunc(grpcServer, httpServer))
	fmt.Println("hello")
}

func grpcHandlerFunc(grpcServer *grpc.Server, httpServer http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			httpServer.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func httpServer() *http.ServeMux {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`pong`))
	})

	return serveMux
}

func grpcServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterSearchServiceServer(server, &SearchService{})
	reflection.Register(server)
	return server
}

func grpcGateway() *runtime.ServeMux {
	//endpoint := "0.0.0.0:" + port
	gwmux := runtime.NewServeMux()
	dopts := []grpc.DialOption{grpc.WithInsecure()}
	//将 http 转成 rpc 请求
	_ = pb.RegisterSearchServiceHandlerFromEndpoint(context.Background(), gwmux, ENDPOINT, dopts)
	return gwmux
}


