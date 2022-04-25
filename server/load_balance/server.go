package main

import (
	"context"
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

var addrs = []string{":9004", ":9005"}

//SearchService 定义服务，需实现了SearchServiceServer，这样该服务才能注册
type SearchService struct{
	addr string
}

//Search 实现了SearchServiceServer接口定义的方法,实现该方法就能填充我们的业务逻辑
func (s *SearchService) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	log.Println("the server port is ", s.addr)
	return &pb.SearchResponse{Response: r.GetRequest() + " Server"}, nil
}

//startServer 接收一个端口起一个服务
func startServer(addr string) {
	server := grpc.NewServer()
	pb.RegisterSearchServiceServer(server, &SearchService{addr})
	log.Printf("serving on %s\n", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	server.Serve(lis)
}

//负载均衡
func main() {
	var wg sync.WaitGroup
	for _, addr := range addrs {
		wg.Add(1)
		go func(val string) {
			defer wg.Done()
			startServer(val)
		}(addr)
	}
	wg.Wait()
}



