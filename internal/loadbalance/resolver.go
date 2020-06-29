package loadbalance

import (
	"context"
	"fmt"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"

	api "github.com/weak-head/proglog/api/v1"
)

const (
	Name = "proglog"
)

type Resolver struct {
	mu            sync.Mutex
	clientConn    resolver.ClientConn
	resolverConn  *grpc.ClientConn
	serviceConfig *serviceconfig.ParseResult
}

var _ resolver.Builder = (*Resolver)(nil)

func (r *Resolver) Build(
	target resolver.Target,
	cc resolver.ClientConn,
	opts resolver.BuildOptions,
) (resolver.Resolver, error) {
	r.clientConn = cc
	var dialOpts []grpc.DialOption

	if opts.DialCreds != nil {
		dialOpts = append(
			dialOpts,
			grpc.WithTransportCredentials(opts.DialCreds),
		)
	}

	// Use 'proglog' load balancer
	r.serviceConfig = r.clientConn.ParseServiceConfig(
		fmt.Sprintf(
			`{"loadBalancingConfig":[{"%s":{}}]}`,
			Name,
		),
	)

	var err error
	r.resolverConn, err = grpc.Dial(target.Endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	r.ResolveNow(resolver.ResolveNowOptions{})
	return r, nil
}

func (r *Resolver) Scheme() string {
	return Name
}

func init() {
	resolver.Register(&Resolver{})
}

var _ resolver.Resolver = (*Resolver)(nil)

func (r *Resolver) ResolveNow(
	resolver.ResolveNowOptions,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := api.NewLogClient(r.resolverConn)
	ctx := context.Background()

	res, err := client.GetServers(ctx, &api.GetServersRequest{})
	if err != nil {
		log.Printf(
			"[ERROR] proglog: failed to resolve servers: %v",
			err,
		)
		return
	}

	var addrs []resolver.Address
	for _, server := range res.Servers {
		addrs = append(
			addrs,
			resolver.Address{
				Addr: server.RpcAddr,
				Attributes: attributes.New(
					"is_leader",
					server.IsLeader,
				),
			},
		)
	}

	// Use 'proglog' load balancer to
	// pick a server from the 'addrs'
	r.clientConn.UpdateState(resolver.State{
		Addresses:     addrs,
		ServiceConfig: r.serviceConfig,
	})
}

func (r *Resolver) Close() {
	if err := r.resolverConn.Close(); err != nil {
		log.Printf("[ERROR] proglog: failed to close conn: %v", err)
	}
}
