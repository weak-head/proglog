package loadbalance

import (
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"

	"github.com/stretchr/testify/require"

	api "github.com/weak-head/proglog/api/v1"
	"github.com/weak-head/proglog/internal/config"
	"github.com/weak-head/proglog/internal/server"
)

func TestResolver(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// Server TLS config
	tlsConfig, err := config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.ServerCertFile,
			KeyFile:       config.ServerKeyFile,
			CAFile:        config.CAFile,
			Server:        true,
			ServerAddress: "127.0.0.1",
		},
	)
	require.NoError(t, err)

	// Server TLS credentials
	serverCreds := credentials.NewTLS(tlsConfig)

	// gRPC server with TLS credentials
	srv, err := server.NewGRPCServer(
		&server.Config{
			// Mock object
			GetServerer: &getServers{},
		},
		grpc.Creds(serverCreds),
	)
	require.NoError(t, err)

	// Serve gRPC server on the localhost
	go srv.Serve(l)

	// Client TLS config
	tlsConfig, err = config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.RootClientCertFile,
			KeyFile:       config.RootClientKeyFile,
			CAFile:        config.CAFile,
			Server:        false,
			ServerAddress: "127.0.0.1",
		},
	)
	require.NoError(t, err)

	// Client TLS credentials
	clientCreds := credentials.NewTLS(tlsConfig)

	// Builder config
	opts := resolver.BuildOptions{
		DialCreds: clientCreds,
	}

	// Mock object
	conn := &clientConn{}

	// Build and configure gRPC resolver
	// to the target endpoint of the server
	// The resolver will call 'GetServers' to resolve
	// the servers and update the client connection
	// with the servers' addresses.
	r := &Resolver{}
	_, err = r.Build(
		resolver.Target{
			Endpoint: l.Addr().String(),
		},
		conn,
		opts,
	)
	require.NoError(t, err)

	// Make sure that states match
	wantState := resolver.State{
		Addresses: []resolver.Address{
			{
				Addr:       "localhost:9001",
				Attributes: attributes.New("is_leader", true),
			},
			{
				Addr:       "localhost:9002",
				Attributes: attributes.New("is_leader", false),
			},
		},
	}
	require.Equal(t, wantState, conn.state)

	// States should match afther the 'ResolveNow'
	conn.state.Addresses = nil
	r.ResolveNow(resolver.ResolveNowOptions{})
	require.Equal(t, wantState, conn.state)
}

type getServers struct{}

func (s *getServers) GetServers() ([]*api.Server, error) {
	return []*api.Server{
		{
			Id:       "leader",
			RpcAddr:  "localhost:9001",
			IsLeader: true,
		},
		{
			Id:      "follower",
			RpcAddr: "localhost:9002",
		},
	}, nil
}

type clientConn struct {
	resolver.ClientConn
	state resolver.State
}

func (c *clientConn) UpdateState(state resolver.State) {
	c.state = state
}

func (c *clientConn) ReportError(err error)                                       {}
func (c *clientConn) NewAddress(addr []resolver.Address)                          {}
func (c *clientConn) NewServiceConfig(config string)                              {}
func (c *clientConn) ParseServiceConfig(config string) *serviceconfig.ParseResult { return nil }
