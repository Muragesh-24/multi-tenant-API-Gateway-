package main

import (
	"context"
	"log"
	"net"

	"engigrow-iam-gateway/proto/iam"
	"google.golang.org/grpc"
)

type iamServer struct {
	iam.UnimplementedIAMServiceServer
}

func (s *iamServer) VerifyAPIKey(ctx context.Context, req *iam.VerifyKeyRequest) (*iam.VerifyKeyResponse, error) {
	log.Printf("Received key verification request for Tenant: %s", req.TenantId)

	// Mocking a validation check for the MVP step
	if req.ApiKey == "engigrow_secret_prod_key" {
		return &iam.VerifyKeyResponse{
			IsValid:    true,
			ClientTier: "premium",
		}, nil
	}

	return &iam.VerifyKeyResponse{
		IsValid:    false,
		ClientTier: "unauthorized",
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	iam.RegisterIAMServiceServer(grpcServer, &iamServer{})

	log.Println("Internal IAM Microservice running smoothly on port :50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}