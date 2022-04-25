package main

import (
	"context"
	"fmt"
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/resolver"
	"log"
)

var addrs = []string{"localhost:9004", "localhost:9005"}

const (
	exampleScheme      = "example"
	exampleServiceName = "lb.example.com"
)

func main() {
	conn, err := grpc.Dial(
		fmt.Sprintf("%s:///%s", exampleScheme, exampleServiceName),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}
	defer conn.Close()

	makeRPCs(conn, 10)
}

func makeRPCs(cc *grpc.ClientConn, n int) {
	client := pb.NewSearchServiceClient(cc)
	for i := 0; i < n; i++ {
		resp, err := client.Search(context.Background(), &pb.SearchRequest{
			Request: "grpc client load balance",
		})
		log.Printf("resp:%v, err:%v", resp, err)
	}
}

type exampleResolverBuilder struct{}

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &exampleResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			exampleServiceName: addrs,
		},
	}

	r.start()
	return r, nil
}

func (*exampleResolverBuilder) Scheme() string { return exampleScheme }

func (r *exampleResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*exampleResolver) Close()                                  {}

func init() {
	resolver.Register(&exampleResolverBuilder{})
}


