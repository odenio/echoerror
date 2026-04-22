package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/example/echoerror/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedEchoErrorServer
}

func (s *server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	grpc.SetTrailer(ctx, metadata.Pairs("x-echo-message", req.Message))

	if req.PadMessageKb > 0 {
		pad := make([]byte, int(req.PadMessageKb)*1024)
		rand.Read(pad)
		grpc.SetTrailer(ctx, metadata.Pairs("x-echo-pad", base64.StdEncoding.EncodeToString(pad)))
	}

	code := codes.Code(req.Code)
	if code == codes.OK {
		return &pb.EchoResponse{}, nil
	}
	log.Printf("returning code=%d message=%q", req.Code, req.Message)
	return nil, status.Errorf(code, "%s", req.Message)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func serve(port, label string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen on %s port %s: %v", label, port, err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoErrorServer(s, &server{})
	fmt.Printf("server listening on :%s (%s)\n", port, label)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve on %s port %s: %v", label, port, err)
	}
}

func main() {
	unmeshedPort := envOrDefault("UNMESHED_LISTEN_PORT", "9000")
	meshedPort := envOrDefault("MESHED_LISTEN_PORT", "9090")

	go serve(unmeshedPort, "unmeshed")
	serve(meshedPort, "meshed")
}
