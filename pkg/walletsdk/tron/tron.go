package tron

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Config struct {
	NodeAddr                  string
	UseTLS                    bool
	ActivationContractAddress string
	UseBurnTRXActivation      bool
	GRPCOptions               []grpc.DialOption
}

type Tron struct {
	node *client.GrpcClient
	conf Config
}

func NewTron(conf Config) (*Tron, error) {
	if conf.NodeAddr == "" {
		return nil, fmt.Errorf("node gRPC address must not be empty")
	}

	return &Tron{
		node: client.NewGrpcClientWithTimeout(conf.NodeAddr, time.Second*30),
		conf: conf,
	}, nil
}

// Name returns the service name
func (t *Tron) Name() string { return "tron-service" }

// Node returns the grpc client
func (t *Tron) Node() *client.GrpcClient { return t.node }

// Start
func (t *Tron) Start(_ context.Context) error {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024 * 1024 * 100)),
	}

	opts = append(opts, t.conf.GRPCOptions...)

	if t.conf.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if t.node != nil {
		if err := t.node.Start(opts...); err != nil {
			return fmt.Errorf("failed to init tron grpc connecton: %w", err)
		}
	}

	return nil
}

// Stop
func (t *Tron) Stop(_ context.Context) error {
	if t.node == nil {
		return nil
	}

	t.node.Stop()

	return nil
}

func PrepareUnaryInterceptor(kv ...string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func PrepareStreamInterceptor(kv ...string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = metadata.AppendToOutgoingContext(ctx, kv...)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// CheckIsWalletActivated checks if the wallet is activated
func (t *Tron) CheckIsWalletActivated(address string) (bool, error) {
	_, err := t.node.GetAccount(address)
	if err == nil {
		return true, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "account not found") {
		return false, nil
	}
	return false, err
}
