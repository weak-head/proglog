package discovery

import (
	"log"
	"net"

	"github.com/hashicorp/serf/serf"
)

type Handler interface {
	Join(name, addr string) error
	Leave(name, addr string) error
}

// Membership is the wrapper around Serf
type Membership struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
}

// New creates a new instance of Membership
func New(handler Handler, config Config) (*Membership, error) {
	c := &Membership{
		Config:  config,
		handler: handler,
	}
	if err := c.setupSerf(); err != nil {
		return nil, err
	}
	return c, nil
}

// Config defines the configuration of the node
type Config struct {
	NodeName       string
	BindAddr       string
	Tags           map[string]string
	StartJoinAddrs []string
}

// setupSerf setups and bootstraps Serf connection
func (m *Membership) setupSerf() (err error) {
	// Resolve IP and port
	addr, err := net.ResolveTCPAddr("tcp", m.BindAddr)
	if err != nil {
		return err
	}

	// Channel to accept Serf events
	m.events = make(chan serf.Event)

	// Configure Serf
	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	config.EventCh = m.events
	config.Tags = m.Tags
	config.NodeName = m.Config.NodeName

	// Create Serf
	m.serf, err = serf.Create(config)
	if err != nil {
		return err
	}

	// Bootstrap even handling
	go m.eventHandler()

	// Join the existing cluster
	if m.StartJoinAddrs != nil {
		_, err = m.serf.Join(m.StartJoinAddrs, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Membership) eventHandler() {
	for e := range m.events {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					continue
				}
				m.handleJoin(member)
			}

		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, member := range e.(serf.MemberEvent).Members {
				if m.isLocal(member) {
					return
				}
				m.handleLeave(member)
			}
		}
	}
}

func (m *Membership) handleJoin(member serf.Member) {
	if err := m.handler.Join(
		member.Name,
		member.Tags["rpc_addr"],
	); err != nil {
		log.Printf(
			"[ERROR] proglog: failed to join: %s %s",
			member.Name,
			member.Tags["rpc_addr"],
		)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	if err := m.handler.Leave(
		member.Name,
		member.Tags["rpc_addr"],
	); err != nil {
		log.Printf(
			"[ERROR] proglog: failed to leave: %s %s",
			member.Name,
			member.Tags["rpc_addr"],
		)
	}
}

func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

// Members returns the list of all members in the cluster
func (m *Membership) Members() []serf.Member {
	return m.serf.Members()
}

// Leave leaves the cluster
func (m *Membership) Leave() error {
	return m.serf.Leave()
}
