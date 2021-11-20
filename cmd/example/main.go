package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	grpcviewer "github.com/fizx/grpc-viewer"
	"github.com/fizx/grpc-viewer/example"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/protobuf/encoding/protojson"
)

type alternateServer struct {
	example.UnimplementedAlternateServiceServer
}

type exampleServer struct {
	example.UnimplementedExampleServiceServer
}

func (s *alternateServer) ExampleMethod(ctx context.Context, in *example.ExampleRequest) (*example.ExampleResponse, error) {
	return &example.ExampleResponse{
		Message: protojson.Format(in) + " (alt)",
	}, nil
}

func (s *exampleServer) ExampleMethod1(ctx context.Context, in *example.ExampleRequest) (*example.ExampleResponse, error) {
	return &example.ExampleResponse{
		Message: protojson.Format(in) + " (1)",
	}, nil
}

func (s *exampleServer) ExampleMethod2(ctx context.Context, in *example.ExampleRequest) (*example.ExampleResponse, error) {
	return &example.ExampleResponse{
		Message: protojson.Format(in) + " (2)",
	}, nil
}

var ex example.ExampleServiceServer = &exampleServer{}
var alt example.AlternateServiceServer = &alternateServer{}

func main() {
	port := 9090
	grpcServer := grpcviewer.NewServer()
	grpcServer.RegisterService(&example.ExampleService_ServiceDesc, ex)
	grpcServer.RegisterService(&example.AlternateService_ServiceDesc, alt)
	grpclog.SetLogger(log.New(os.Stdout, "example: ", log.LstdFlags))

	handler := func(resp http.ResponseWriter, req *http.Request) {
		grpcServer.ServeHTTP(resp, req)
	}

	httpServer := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handler),
	}
	grpclog.Printf("Starting server. http port: %d, with TLS: %v", port, false)
	if err := httpServer.ListenAndServe(); err != nil {
		grpclog.Fatalf("failed starting http server: %v", err)
	}
}
