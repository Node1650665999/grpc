package main

import (
	"context"
	"io"
	"log"

	"google.golang.org/grpc"

	pb "go-grpc/proto"
)

const (
	PORT = "9002"
)

func main() {
	conn, err := grpc.Dial(":"+PORT, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc.Dial err: %v", err)
	}

	defer conn.Close()

	client := pb.NewStreamServiceClient(conn)

	/*err = printServerSideStream(client, &pb.StreamRequest{Pt: &pb.StreamPoint{Name: "gRPC Send ServerSide stream", Value: 2018}})
	if err != nil {
		log.Fatalf("printServerSideStream.err: %v", err)
	}*/

	/*err = printClientSideStream(client, &pb.StreamRequest{Pt: &pb.StreamPoint{Name: "gRPC Send ClientSide  stream", Value: 2018}})
	if err != nil {
		log.Fatalf("printClientSideStream.err: %v", err)
	}*/

	err = printBidStream(client, &pb.StreamRequest{Pt: &pb.StreamPoint{Name: "gRPC send Bidirectional stream", Value: 2018}})
	if err != nil {
		log.Fatalf("printBidStream.err: %v", err)
	}
}

func printServerSideStream(client pb.StreamServiceClient, r *pb.StreamRequest) error {
	stream, err := client.ServerSideStream(context.Background(), r)
	if err != nil {
		return err
	}

	for {
		//接收数据流
		resp, err := stream.Recv()
		//当流成功/结束（调用了 Close）时,会返回 io.EOF
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		log.Printf("resp: pj.name: %s, pt.value: %d", resp.Pt.Name, resp.Pt.Value)
	}

	return nil
}

func printClientSideStream(client pb.StreamServiceClient, r *pb.StreamRequest) error {
	stream, err := client.ClientSideStream(context.Background())
	if err != nil {
		return err
	}

	tmp := r.Pt.Value
	for n := 0; n <= 6; n++ {
		//发送数据流
		r.Pt.Value = tmp + int32(n)
		err := stream.Send(r)
		if err != nil {
			return err
		}
	}

	//告诉对端流结束
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Printf("resp: pj.name: %s, pt.value: %d", resp.Pt.Name, resp.Pt.Value)

	return nil
}

func printBidStream(client pb.StreamServiceClient, r *pb.StreamRequest) error {
	stream, err := client.BidStream(context.Background())
	if err != nil {
		return err
	}

	for n := 0; n <= 6; n++ {
		//流式发送
		err = stream.Send(r)
		if err != nil {
			return err
		}

		//流式接收
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		log.Printf("resp: pj.name: %s, pt.value: %d", resp.Pt.Name, resp.Pt.Value)
	}

	//告诉对端流结束
	stream.CloseSend()

	return nil
}