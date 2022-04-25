package main

import (
	"context"
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"log"
	"time"
)

const PORT = "9001"

//go 1.15 版本开始废弃 CommonName 方式生成的证书，因此推荐使用 SAN 证书。
//如果想兼容之前的方式，需要设置环境变量 GODEBUG 为 x509ignoreCN=0,
//即运行的命令为: GODEBUG="x509ignoreCN=0" go run client.go
func main() {
	conn, err := grpc.Dial(":"+PORT, grpc.WithInsecure())
	defer conn.Close()
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}

	client := pb.NewSearchServiceClient(conn)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5 * time.Second))
	defer cancel()
	resp, err := client.Search(ctx, &pb.SearchRequest{
		Request: "gRPC",
	})
	if err != nil {
		log.Fatalf("client.Search err: %v", err)
	}

	log.Printf("resp: %s", resp.GetResponse())
}



