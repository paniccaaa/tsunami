// Mock gRPC server for testing Tsunami load testing tool.
//
// Usage:
//
//	go run ./examples/grpc-server --port 50051 --reflection        # with reflection (default)
//	go run ./examples/grpc-server --port 50051 --reflection=false  # without reflection (requires proto file)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"

	pb "github.com/paniccaaa/tsunami/examples/grpc-server/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type greeterServer struct {
	pb.UnimplementedGreeterServiceServer
	requestCount atomic.Int64
	errCount     atomic.Int64
}

func (s *greeterServer) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	id := s.requestCount.Add(1)
	name := req.GetName()
	if name == "" {
		name = "stranger"
	}
	return &pb.HelloReply{
		Message:   fmt.Sprintf("Hello, %s!", name),
		RequestId: id,
	}, nil
}

func (s *greeterServer) Ping(_ context.Context, _ *pb.Empty) (*pb.PingReply, error) {
	return &pb.PingReply{
		Status:      "ok",
		TimestampMs: time.Now().UnixMilli(),
	}, nil
}

// loggingInterceptor logs every RPC: method, peer, latency, status code.
// It also prints a rolling summary every statsInterval requests.
func loggingInterceptor(statsInterval int64) grpc.UnaryServerInterceptor {
	var total, errs atomic.Int64

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		latency := time.Since(start)

		n := total.Add(1)
		code := codes.OK
		if err != nil {
			code = status.Code(err)
			errs.Add(1)
		}

		// Per-request log (only first few, then throttle to summary only)
		if n <= 5 {
			src := "unknown"
			if p, ok := peer.FromContext(ctx); ok {
				src = p.Addr.String()
			}
			log.Printf("[rpc] %-40s  %s  %v  from %s", info.FullMethod, code, latency.Round(time.Microsecond), src)
		} else if n == 6 {
			log.Printf("[rpc] (per-request logs suppressed, printing summary every %d requests)", statsInterval)
		}

		// Rolling summary
		if n%statsInterval == 0 {
			e := errs.Load()
			successRate := float64(n-e) / float64(n) * 100
			log.Printf("[stats] total=%-8d  errors=%-6d  success=%.1f%%", n, e, successRate)
		}

		return resp, err
	}
}

func main() {
	port := flag.Int("port", 50051, "Port to listen on")
	withReflection := flag.Bool("reflection", true, "Enable gRPC server reflection (set false to test proto-file mode)")
	statsEvery := flag.Int64("stats-every", 1000, "Print summary stats every N requests")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(*statsEvery)),
	)
	pb.RegisterGreeterServiceServer(srv, &greeterServer{})

	if *withReflection {
		reflection.Register(srv)
		log.Printf("reflection: enabled  (leave Proto File empty in Tsunami)")
	} else {
		log.Printf("reflection: disabled (upload examples/grpc-server/greeter.proto in Tsunami)")
	}

	log.Printf("stats summary: every %d requests (--stats-every to change)", *statsEvery)
	log.Printf("listening on :%d", *port)
	log.Printf("")
	log.Printf("Test config:")
	log.Printf("  Target  : localhost:%d", *port)
	log.Printf("  Service : greeter.GreeterService")
	log.Printf("  Method  : SayHello   payload: {\"name\": \"world\"}")
	log.Printf("  Method  : Ping       payload: {}")

	if err := srv.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
