package grpcattack

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/paniccaaa/tsunami/internal/attack"
)

func RunAttack(cfg *Config, stopCh chan struct{}, metrics *attack.GlobalMetrics) (*attack.GlobalMetrics, time.Duration, error) {
	if metrics == nil {
		metrics = attack.NewGlobalMetrics()
	}

	startTime := time.Now()
	metrics.SetTestConfig(cfg.RPS, cfg.Duration, startTime)

	poolSize := int(cfg.Connections)
	if poolSize == 0 {
		poolSize = 4
	}

	dialOpt, err := buildDialOption(cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("build dial options: %w", err)
	}

	conns := make([]*grpc.ClientConn, poolSize)
	for i := range conns {
		conn, err := grpc.NewClient(cfg.Target, dialOpt)
		if err != nil {
			for j := 0; j < i; j++ {
				conns[j].Close()
			}
			return nil, 0, fmt.Errorf("dial %q: %w", cfg.Target, err)
		}
		conns[i] = conn
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	methodDesc, err := ResolveMethod(cfg, conns[0])
	if err != nil {
		return nil, 0, fmt.Errorf("resolve method: %w", err)
	}

	md := parseMetadata(cfg.Metadata)

	workerCount := int(cfg.Workers)
	bufSize := workerCount * 2
	if bufSize < 100 {
		bufSize = 100
	}
	jobs := make(chan struct{}, bufSize)
	results := make(chan attack.RequestResult, workerCount*2)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		conn := conns[i%poolSize]
		go grpcWorker(conn, cfg, methodDesc, md, jobs, results, &wg)
	}

	var stopOnce sync.Once
	if cfg.Duration > 0 {
		go func() {
			select {
			case <-time.After(cfg.Duration):
				stopOnce.Do(func() { close(stopCh) })
			case <-stopCh:
			}
		}()
	}

	rl := ratelimit.New(cfg.RPS)
	go func() {
		for {
			rl.Take()
			select {
			case <-stopCh:
				close(jobs)
				return
			case jobs <- struct{}{}:
			}
		}
	}()

	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		for r := range results {
			metrics.Lock()
			metrics.TotalRequests++
			if r.Success {
				metrics.Successes++
			} else {
				metrics.Failures++
				metrics.AddError(r.ErrorType)
			}
			metrics.TotalLatency += r.Latency
			metrics.AddLatency(r.Latency)
			metrics.StatusCodes[r.StatusCode]++
			metrics.Unlock()
		}
	}()

	<-stopCh
	wg.Wait()
	close(results)
	collectorWg.Wait()

	elapsed := time.Since(startTime)
	if cfg.Duration > 0 && elapsed > cfg.Duration {
		elapsed = cfg.Duration
	}
	return metrics, elapsed, nil
}

func grpcWorker(
	conn *grpc.ClientConn,
	cfg *Config,
	methodDesc protoreflect.MethodDescriptor,
	md metadata.MD,
	jobs <-chan struct{},
	results chan<- attack.RequestResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	data := cfg.Data
	if data == "" {
		data = "{}"
	}

	fullMethod := "/" + string(methodDesc.Parent().FullName()) + "/" + string(methodDesc.Name())

	for range jobs {
		req := dynamicpb.NewMessage(methodDesc.Input())
		if err := protojson.Unmarshal([]byte(data), req); err != nil {
			results <- attack.RequestResult{
				Success:   false,
				ErrorType: attack.ErrorTypeOther,
			}
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		if len(md) > 0 {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		resp := dynamicpb.NewMessage(methodDesc.Output())
		start := time.Now()
		err := conn.Invoke(ctx, fullMethod, req, resp)
		latency := time.Since(start)
		cancel()

		results <- buildResult(err, latency)
	}
}

func buildResult(err error, latency time.Duration) attack.RequestResult {
	if err == nil {
		return attack.RequestResult{
			StatusCode: int(codes.OK),
			Latency:    latency,
			Success:    true,
		}
	}
	code := status.Code(err)
	return attack.RequestResult{
		StatusCode: int(code),
		Latency:    latency,
		Success:    false,
		ErrorType:  grpcCodeToErrorType(code),
	}
}

func grpcCodeToErrorType(code codes.Code) attack.ErrorType {
	switch code {
	case codes.DeadlineExceeded:
		return attack.ErrorTypeTimeout
	case codes.Unavailable:
		return attack.ErrorTypeConnectionRefused
	default:
		return attack.ErrorTypeOther
	}
}

func buildDialOption(cfg *Config) (grpc.DialOption, error) {
	if cfg.Insecure {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}
	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read CA cert %q: %w", cfg.CACert, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to parse CA cert %q", cfg.CACert)
		}
		return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{RootCAs: pool})), nil
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})), nil
}

func parseMetadata(pairs []string) metadata.MD {
	md := metadata.MD{}
	for _, p := range pairs {
		idx := strings.IndexByte(p, ':')
		if idx <= 0 {
			continue
		}
		md.Append(strings.TrimSpace(p[:idx]), strings.TrimSpace(p[idx+1:]))
	}
	return md
}
