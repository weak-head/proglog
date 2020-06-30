package agent

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	api "github.com/weak-head/proglog/api/v1"
	"github.com/weak-head/proglog/internal/config"
	"github.com/weak-head/proglog/internal/loadbalance"
)

func TestAgent(t *testing.T) {
	// Configuration of the certificate that is served
	// to the clients
	serverTLSConfig, err := config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.ServerCertFile,
			KeyFile:       config.ServerKeyFile,
			CAFile:        config.CAFile,
			Server:        true,
			ServerAddress: "127.0.0.1",
		},
	)
	require.NoError(t, err)

	// Configuration of the certificate that is served
	// between servers, so they can connect with and
	// replicate each other
	peerTLSConfig, err := config.SetupTLSConfig(
		config.TLSConfig{
			CertFile:      config.RootClientCertFile,
			KeyFile:       config.RootClientKeyFile,
			CAFile:        config.CAFile,
			Server:        false,
			ServerAddress: "127.0.0.1",
		},
	)
	require.NoError(t, err)

	// Setup up the cluster of 3 nodes
	var agents []*Agent
	for i := 0; i < 3; i++ {
		// Allocate two ports for each service:
		// - gRPC log connection
		// - Serf service discovery connection
		ports := dynaport.Get(2)
		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
		rpcPort := ports[1]

		dataDir, err := ioutil.TempDir(os.TempDir(), "server_test_log")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(
				startJoinAddrs,
				agents[0].Config.BindAddr,
			)
		}

		agent, err := New(
			Config{
				NodeName:        fmt.Sprintf("%d", i),
				StartJoinAddr:   startJoinAddrs,
				BindAddr:        bindAddr,
				RPCPort:         rpcPort,
				DataDir:         dataDir,
				ACLModelFile:    config.ACLModelFile,
				ACLPolicyFile:   config.ACLPolicyFile,
				ServerTLSConfig: serverTLSConfig,
				PeerTLSConfig:   peerTLSConfig,
				Bootstrap:       i == 0,
			},
		)
		require.NoError(t, err)

		agents = append(agents, agent)
	}

	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(
				t,
				os.RemoveAll(agent.Config.DataDir),
			)
		}
	}()

	// Wait until all the nodes are initialized and connected
	// to the cluster
	time.Sleep(3 * time.Second)

	leaderClient := client(t, agents[0], peerTLSConfig)
	produceResponse, err := leaderClient.Produce(
		context.Background(),
		&api.ProduceRequest{
			Record: &api.Record{
				Value: []byte("foo"),
			},
		},
	)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	consumeResponse, err := leaderClient.Consume(
		context.Background(),
		&api.ConsumeRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	// Wait until the replication has finished
	time.Sleep(3 * time.Second)

	followerClient := client(t, agents[1], peerTLSConfig)
	consumeResponse, err = followerClient.Consume(
		context.Background(),
		&api.ConsumeRequest{
			Offset: produceResponse.Offset,
		},
	)
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	// There should be no more messages to consume
	consumeResponse, err = leaderClient.Consume(
		context.Background(),
		&api.ConsumeRequest{
			Offset: produceResponse.Offset + 1,
		},
	)
	require.Nil(t, consumeResponse)
	require.Error(t, err)
	got := grpc.Code(err)
	want := grpc.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	require.Equal(t, got, want)
}

func client(
	t *testing.T,
	agent *Agent,
	tlsConfig *tls.Config,
) api.LogClient {
	// Client TLS config
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}

	// gRPC server endpoint
	rpcAddr, err := agent.Config.RPCAddr()
	require.NoError(t, err)

	// dial gRPC server with 'proglog' load balancer
	conn, err := grpc.Dial(
		fmt.Sprintf(
			"%s:///%s",
			loadbalance.Name,
			rpcAddr,
		),
		opts...,
	)
	require.NoError(t, err)

	// Create new Log client
	return api.NewLogClient(conn)
}
