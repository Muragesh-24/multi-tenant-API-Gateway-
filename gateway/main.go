package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

   "engigrow-iam-gateway/proto/iam"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Gateway struct {
	iamClient iam.IAMServiceClient
	rdb       *redis.Client
}

// isRateLimited implements a Leaky Bucket algorithm using Redis
func (g *Gateway) isRateLimited(ctx context.Context, tenantID string, tier string) bool {
	// Set bucket rules based on SaaS tier
	var capacity float64 = 10.0 // Max request bursts allowed
	var leakRate float64 = 2.0  // Leaks 2 requests per second

	if tier == "premium" {
		capacity = 50.0 // Premium tiers get bigger buckets
		leakRate = 10.0 // Leaks 10 requests per second
	}

	now := time.Now().UnixNano()
	nowSec := float64(now) / 1e9

	redisKey := fmt.Sprintf("bucket:%s", tenantID)

	// We use a Redis transaction (Pipelining) to read and update water levels atomically
	pipe := g.rdb.TxPipeline()
	waterGet := pipe.HGet(ctx, redisKey, "water")
	timeGet := pipe.HGet(ctx, redisKey, "last_updated")
	_, _ = pipe.Exec(ctx)

	var water float64 = 0.0
	var lastUpdated float64 = nowSec

	if waterStr, err := waterGet.Result(); err == nil {
		water, _ = strconv.ParseFloat(waterStr, 64)
	}
	if timeStr, err := timeGet.Result(); err == nil {
		lastUpdated, _ = strconv.ParseFloat(timeStr, 64)
	}

	// 1. Leak the water based on time passed
	elapsed := nowSec - lastUpdated
	water = water - (elapsed * leakRate)
	if water < 0 {
		water = 0
	}

	// 2. Check if adding this new request overflows the bucket
	if water+1.0 > capacity {
		return true // Overflow! Drop request (Rate Limited)
	}

	// 3. Update the bucket state in Redis
	water += 1.0
	pipe = g.rdb.TxPipeline()
	pipe.HSet(ctx, redisKey, "water", water)
	pipe.HSet(ctx, redisKey, "last_updated", nowSec)
	pipe.Expire(ctx, redisKey, 10*time.Second) // Cleanup stale keys
	_, _ = pipe.Exec(ctx)

	return false
}

func (g *Gateway) handleValidateKey(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	tenantID := r.Header.Get("X-Tenant-ID")

	if apiKey == "" || tenantID == "" {
		http.Error(w, "Missing authentication headers", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// 1. Ultra-low latency gRPC verification with IAM Service
	rpcCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	res, err := g.iamClient.VerifyAPIKey(rpcCtx, &iam.VerifyKeyRequest{
		ApiKey:   apiKey,
		TenantId: tenantID,
	})

	w.Header().Set("Content-Type", "application/json")
	if err != nil || !res.IsValid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "denied", "reason": "Invalid credentials"})
		return
	}

	// 2. Enforce the Leaky Bucket rate limiter based on the tier returned via gRPC
	if g.isRateLimited(ctx, tenantID, res.ClientTier) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "rejected",
			"reason": "SaaS API rate quota exceeded. Please upgrade your tier.",
		})
		return
	}

	// 3. Success response
	json.NewEncoder(w).Encode(map[string]string{
		"status": "authorized",
		"tier":   res.ClientTier,
	})
}

func main() {
	// Initialize Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize internal gRPC connection
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to IAM Microservice: %v", err)
	}
	defer conn.Close()

	gw := &Gateway{
		iamClient: iam.NewIAMServiceClient(conn),
		rdb:       rdb,
	}

	http.HandleFunc("/v1/validate", gw.handleValidateKey)

	log.Println("External API Gateway routing incoming traffic on port :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Gateway server failure: %v", err)
	}
}