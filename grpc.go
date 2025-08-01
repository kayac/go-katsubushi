package katsubushi

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/kayac/go-katsubushi/v2/grpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	MaxGRPCBulkSize = 1000
)

type gRPCGenerator struct {
	grpc.GeneratorServer
	app *App
}

func (sv *gRPCGenerator) Fetch(ctx context.Context, req *grpc.FetchRequest) (*grpc.FetchResponse, error) {
	atomic.AddInt64(&sv.app.cmdGet, 1)
	slog.Debug("gRPC Fetch request")

	id, err := sv.app.NextID()
	if err != nil {
		return nil, fmt.Errorf("failed to get id: %w", err)
	}
	slog.Debug("gRPC Generated ID", "id", id)
	res := &grpc.FetchResponse{
		Id: id,
	}
	return res, nil
}

func (sv *gRPCGenerator) FetchMulti(ctx context.Context, req *grpc.FetchMultiRequest) (*grpc.FetchMultiResponse, error) {
	atomic.AddInt64(&sv.app.cmdGet, 1)
	n := int(req.N)
	slog.Debug("gRPC FetchMulti request", "n", n)
	if n > MaxGRPCBulkSize {
		return nil, fmt.Errorf("too many IDs requested: %d, n should be smaller than %d", n, MaxGRPCBulkSize)
	}
	if n == 0 {
		n = 1
	}
	ids := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		id, err := sv.app.NextID()
		if err != nil {
			return nil, fmt.Errorf("failed to get id: %w", err)
		}
		ids = append(ids, id)
	}
	slog.Debug("gRPC Generated IDs", "ids", ids)
	res := &grpc.FetchMultiResponse{
		Ids: ids,
	}
	return res, nil
}

func (app *App) RunGRPCServer(ctx context.Context, cfg *Config) error {
	svGen := &gRPCGenerator{app: app}
	svStats := &gRPCStats{app: app}

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(grpcRecoveryFunc),
	}
	s := gogrpc.NewServer(grpc_middleware.WithUnaryServerChain(
		grpc_recovery.UnaryServerInterceptor(opts...),
	))
	grpc.RegisterGeneratorServer(s, svGen)
	grpc.RegisterStatsServer(s, svStats)
	
	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	
	// Set health status for overall server and individual services
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(grpc.Generator_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(grpc.Stats_ServiceDesc.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
	
	reflection.Register(s)

	listener := cfg.GRPCListener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}
	listener = app.wrapListener(listener)
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down gRPC server")
		s.Stop()
	}()

	slog.Info("Listening gRPC server at " + listener.Addr().String())
	return s.Serve(listener)
}

func grpcRecoveryFunc(p interface{}) error {
	slog.Error("panic", "value", p)
	return status.Errorf(codes.Internal, "Unexpected error")
}

type gRPCStats struct {
	grpc.StatsServer
	app *App
}

func (sv *gRPCStats) Get(ctx context.Context, req *grpc.StatsRequest) (*grpc.StatsResponse, error) {
	slog.Debug("gRPC Stats request")
	st := sv.app.GetStats()
	return &grpc.StatsResponse{
		Pid:              int32(st.Pid),
		Uptime:           st.Uptime,
		Time:             st.Time,
		Version:          st.Version,
		CurrConnections:  st.CurrConnections,
		TotalConnections: st.TotalConnections,
		CmdGet:           st.CmdGet,
		GetHits:          st.GetHits,
		GetMisses:        st.GetMisses,
	}, nil
}
