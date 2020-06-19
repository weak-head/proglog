package discovery

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
)

func TestMembership(t *testing.T) {
	m, handler := setupMember(t, nil)
	m, _ = setupMember(t, m)
	m, _ = setupMember(t, m)

	// 2 nodes should join the cluster
	// 3 nodes in total
	// 0 nodes should leave
	require.Eventually(t,
		func() bool {
			return 2 == len(handler.joins) &&
				3 == len(m[0].Members()) &&
				0 == len(handler.leaves)
		},
		3*time.Second,
		250*time.Millisecond,
	)

	// 3rd node should leave the cluster
	require.NoError(t, m[2].Leave())

	// 2 nodes should join the cluster
	// 3 nodes in total
	// 1st node should detect that 3rd node has left
	// 1 nodes should leave
	require.Eventually(t,
		func() bool {
			return 2 == len(handler.joins) &&
				3 == len(m[0].Members()) &&
				serf.StatusLeft == m[0].Members()[2].Status &&
				1 == len(handler.leaves)
		},
		3*time.Second,
		250*time.Millisecond,
	)

	// 2nd node should leave the cluster
	require.Equal(t, fmt.Sprintf("%d", 2), <-handler.leaves)
}

func setupMember(t *testing.T, members []*Membership) (
	[]*Membership, *handler,
) {
	id := len(members)
	ports := dynaport.Get(1)
	addr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
	tags := map[string]string{
		"rpc_addr": addr,
	}
	c := Config{
		NodeName: fmt.Sprintf("%d", id),
		BindAddr: addr,
		Tags:     tags,
	}
	h := &handler{}

	if len(members) == 0 {
		h.joins = make(chan map[string]string, 3)
		h.leaves = make(chan string, 3)
	} else {
		c.StartJoinAddrs = []string{
			members[0].BindAddr,
		}
	}

	m, err := New(h, c)
	require.NoError(t, err)

	members = append(members, m)
	return members, h
}

type handler struct {
	joins  chan map[string]string
	leaves chan string
}

func (h *handler) Join(id, addr string) error {
	if h.joins != nil {
		h.joins <- map[string]string{
			"id":   id,
			"addr": addr,
		}
	}
	return nil
}

func (h *handler) Leave(id, addr string) error {
	if h.leaves != nil {
		h.leaves <- id
	}
	return nil
}
