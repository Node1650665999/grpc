package main

import (
	pb "go-grpc/proto"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
)

type StreamService struct{}

const (
	PORT = "9002"
)

func main() {
	server := grpc.NewServer()
	pb.RegisterStreamServiceServer(server, &StreamService{})

	lis, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	server.Serve(lis)
}

//ServerSideStream 服务器端流式RPC
//客户端发起一次普通的 RPC 请求,服务端通过流式响应多次发送数据集,客户端 Recv 接收数据集
func (s *StreamService) ServerSideStream(r *pb.StreamRequest, stream pb.StreamService_ServerSideStreamServer) error {
	for n := 0; n <= 6; n++ {
		//stream.Send 发送数据流
		err := stream.Send(&pb.StreamResponse{
			Pt: &pb.StreamPoint{
				Name:  r.Pt.Name,
				Value: r.Pt.Value + int32(n),
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

//ClientSideStream 客户端流式 RPC
//客户端通过流式发起多次 RPC 请求给服务端,服务端发起一次响应给客户端
func (s *StreamService) ClientSideStream(stream pb.StreamService_ClientSideStreamServer) error {
	for {
		//stream.Recv 来接收数据流
		r, err := stream.Recv()
		//当流成功/结束（调用了 Close）时,会返回 io.EOF
		if err == io.EOF {
			return stream.SendAndClose(&pb.StreamResponse{Pt: &pb.StreamPoint{Name: "gRPC response ClientSide Stream", Value: 1}})
		}
		if err != nil {
			return err
		}

		log.Printf("stream.Recv pt.name: %s, pt.value: %d", r.Pt.Name, r.Pt.Value)
	}

	return nil
}

//BidStream 双向流式 RPC
//由客户端以流式的方式发起请求，服务端同样以流式的方式响应请求
func (s *StreamService) BidStream(stream pb.StreamService_BidStreamServer) error {
	n := 0
	for {
		//发送数据流
		err := stream.Send(&pb.StreamResponse{
			Pt: &pb.StreamPoint{
				Name:  "gRPC response Bid Stream",
				Value: int32(n),
			},
		})
		if err != nil {
			return err
		}

		//接收数据流
		r, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		n++

		log.Printf("stream.Recv pt.name: %s, pt.value: %d", r.Pt.Name, r.Pt.Value)
	}

	return nil
}