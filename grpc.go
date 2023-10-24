package katsubushi

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/kayac/go-katsubushi/v2/grpc"
	"github.com/pkg/errors"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
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

	id, err := sv.app.NextID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get id")
	}
	res := &grpc.FetchResponse{
		Id: id,
	}
	return res, nil
}

func (sv *gRPCGenerator) FetchMulti(ctx context.Context, req *grpc.FetchMultiRequest) (*grpc.FetchMultiResponse, error) {
	atomic.AddInt64(&sv.app.cmdGet, 1)
	n := int(req.N)
	if n > MaxGRPCBulkSize {
		return nil, errors.Errorf("too many IDs requested: %d, n should be smaller than %d", n, MaxGRPCBulkSize)
	}
	if n == 0 {
		n = 1
	}
	ids := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		id, err := sv.app.NextID()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get id")
		}
		ids = append(ids, id)
	}
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
	reflection.Register(s)

	listener := cfg.GRPCListener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			return errors.Wrap(err, "failed to listen")
		}
	}
	listener = app.wrapListener(listener)
	go func() {
		<-ctx.Done()
		log.Infof("Shutting down gRPC server")
		s.Stop()
	}()

	log.Infof("Listening gRPC server at %s", listener.Addr())
	return s.Serve(listener)
}

func grpcRecoveryFunc(p interface{}) error {
	log.Errorf("panic: %v", p)
	return status.Errorf(codes.Internal, "Unexpected error")
}

type gRPCStats struct {
	grpc.StatsServer
	app *App
}

func (sv *gRPCStats) Get(ctx context.Context, req *grpc.StatsRequest) (*grpc.StatsResponse, error) {
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

type GRPCClient struct {
	client grpc.GeneratorClient
}

func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := gogrpc.Dial(
		addr,
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
		gogrpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	return &GRPCClient{
		client: grpc.NewGeneratorClient(conn),
	}, nil
}

func (g *GRPCClient) Fetch(ctx context.Context) (uint64, error) {
	res, err := g.client.Fetch(ctx, &grpc.FetchRequest{})
	if err != nil {
		return 0, err
	}
	return res.Id, nil
}

func (g *GRPCClient) FetchMulti(ctx context.Context, n int) ([]uint64, error) {
	res, err := g.client.FetchMulti(ctx, &grpc.FetchMultiRequest{N: uint32(n)})
	if err != nil {
		return nil, err
	}
	return res.Ids, nil
}
