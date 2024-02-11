package quicpunch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/pion/ice/v2"
	"github.com/pion/stun"
)

var (
	urls = []*stun.URI{
		{
			Scheme: stun.SchemeTypeSTUN,
			Host:   "stun.l.google.com",
			Port:   19302,
		},
	}
	networkTypes = []ice.NetworkType{
		ice.NetworkTypeUDP4,
		ice.NetworkTypeUDP6,
	}
)

type session struct {
	Candidates []string

	LocalFrag string
	LocalPwd  string

	RemoteFrag string
	RemotePwd  string

	TrustCert string // PEM
}

func genCreds() (string, string, error) {
	frag := make([]byte, 5)
	if _, err := rand.Read(frag); err != nil {
		return "", "", err
	}

	pwd := make([]byte, 16)
	if _, err := rand.Read(pwd); err != nil {
		return "", "", err
	}

	return hex.EncodeToString(frag), hex.EncodeToString(pwd), nil
}

func startAgent(ctx context.Context, localUfrag, localPwd string, waitForCandidates bool) (*ice.Agent, error) {
	config := &ice.AgentConfig{
		Urls:         urls,
		LocalUfrag:   localUfrag,
		LocalPwd:     localPwd,
		NetworkTypes: networkTypes,
	}

	agent, err := ice.NewAgent(config)
	if err != nil {
		return nil, err
	}

	_ = agent.OnConnectionStateChange(func(cs ice.ConnectionState) {
		fmt.Printf("connection state changed: %s\n", cs)
	})
	_ = agent.OnSelectedCandidatePairChange(func(c1, c2 ice.Candidate) {
		fmt.Printf("selected pair changed: %s <-> %s\n", c1, c2)
	})

	gatheringDone := make(chan struct{})
	_ = agent.OnCandidate(func(c ice.Candidate) {
		if c == nil {
			close(gatheringDone)
		}
	})

	if err := agent.GatherCandidates(); err != nil {
		return nil, err
	}

	if waitForCandidates {
		select {
		case <-ctx.Done():
		case <-gatheringDone:
		}
	}

	return agent, nil
}
