package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/LaboroOptimus/grinex/internal/client/grinex"
	"github.com/LaboroOptimus/grinex/internal/service"
	transportgrpc "github.com/LaboroOptimus/grinex/internal/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	defaultAddr := getenv("GRPC_ADDR", ":50051")
	grpcAddr := flag.String("grpc-addr", defaultAddr, "gRPC listen address")
	flag.Parse()

	listener, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", *grpcAddr, err)
	}

	grinexClient := grinex.NewClient()
	ratesService := service.NewRatesService(grinexClient)
	ratesServer := transportgrpc.NewServer(ratesService)

	grpcServer := grpc.NewServer()
	transportgrpc.Register(grpcServer, ratesServer)
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server listening on %s", *grpcAddr)
		if serveErr := grpcServer.Serve(listener); serveErr != nil {
			log.Printf("gRPC server stopped with error: %v", serveErr)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Printf("shutting down gRPC server")
	grpcServer.GracefulStop()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
