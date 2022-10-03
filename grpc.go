package katsubushi

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/kayac/go-katsubushi/grpc"
	"github.com/pkg/errors"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	MaxGRPCBulkSize = 1000
)

type gRPCServer struct {
	grpc.GeneratorServer
	app *App
}

func (s *gRPCServer) Fetch(ctx context.Context, req *grpc.FetchRequest) (*grpc.FetchResponse, error) {
	atomic.AddInt64(&s.app.cmdGet, 1)

	id, err := s.app.NextID()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get id")
	}
	res := &grpc.FetchResponse{
		Id: id,
	}
	return res, nil
}

func (s *gRPCServer) FetchMulti(ctx context.Context, req *grpc.FetchMultiRequest) (*grpc.FetchMultiResponse, error) {
	atomic.AddInt64(&s.app.cmdGet, 1)
	n := int(req.N)
	if n > MaxGRPCBulkSize {
		return nil, errors.Errorf("too many IDs requested: %d, n should be smaller than %d", n, MaxGRPCBulkSize)
	}
	if n == 0 {
		n = 1
	}
	ids := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		id, err := s.app.NextID()
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
	sv := &gRPCServer{
		app: app,
	}

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(grpcRecoveryFunc),
	}
	s := gogrpc.NewServer(grpc_middleware.WithUnaryServerChain(
		grpc_recovery.UnaryServerInterceptor(opts...),
	))
	grpc.RegisterGeneratorServer(s, sv)
	reflection.Register(s)

	listener := cfg.GRPCListener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			return errors.Wrap(err, "failed to listen")
		}
	}
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
