package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"time"

	"engigrow-iam-gateway/proto/iam"
	"google.golang.org/grpc"
)

type iamServer struct {
	proto.UnimplementedIAMServiceServer
	privateKey *rsa.PrivateKey
}

// Generate the signature using the private key
func (s *iamServer) signData(payload string) (string, error) {
	hashed := sha256.Sum256([]byte(payload))
	signatureBytes, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signatureBytes), nil
}

func (s *iamServer) VerifyAPIKey(ctx context.Context, req *proto.VerifyKeyRequest) (*proto.VerifyKeyResponse, error) {
	log.Printf("Verifying key for Tenant: %s", req.TenantId)

	if req.ApiKey == "engigrow_secret_prod_key" {
		tier := "premium"
		// Create a timestamped payload to prevent replay attacks
		payload := fmt.Sprintf("tenant:%s|tier:%s|ts:%d", req.TenantId, tier, time.Now().Unix())
		
		signature, err := s.signData(payload)
		if err != nil {
			log.Printf("Error signing data: %v", err)
			return nil, err
		}

		return &proto.VerifyKeyResponse{
			IsValid:       true,
			ClientTier:    tier,
			SignedPayload: payload,
			Signature:     signature,
		}, nil
	}

	return &proto.VerifyKeyResponse{
		IsValid: false,
	}, nil
}

func main() {
	// Generate a fresh RSA-2048 keypair on boot
	log.Println("Generating RSA-2048 Private/Public Keypair...")
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA keys: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterIAMServiceServer(grpcServer, &iamServer{
		privateKey: privKey,
	})

	log.Println("Internal IAM Microservice (Cryptographic Authority) running on :50051...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}