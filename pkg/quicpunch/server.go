package quicpunch

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-slug"
	"github.com/pion/ice/v2"
	"github.com/quic-go/quic-go/http3"
)

func packageServer(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("got workspace request")
		_, err := slug.Pack(dir, w, true)
		if err != nil {
			fmt.Printf("failed to pack workspace: %v\n", err)
			http.Error(w, "failed to pack workspace", http.StatusInternalServerError)
		}
	})
}

func prepareSession(ctx context.Context, sess *session) (*ice.Agent, error) {
	localUfrag, localPwd, err := genCreds()
	if err != nil {
		return nil, err
	}
	remoteUfrag, remotePwd, err := genCreds()
	if err != nil {
		return nil, err
	}

	agent, err := startAgent(ctx, localUfrag, localPwd, true)
	if err != nil {
		return nil, err
	}
	localCandidates, err := agent.GetLocalCandidates()
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, len(localCandidates))
	for _, cand := range localCandidates {
		candidates = append(candidates, cand.Marshal())
	}
	fmt.Println("gathering complete")

	if err := agent.SetRemoteCredentials(remoteUfrag, remotePwd); err != nil {
		return nil, err
	}

	sess.Candidates = candidates
	sess.LocalFrag = localUfrag
	sess.LocalPwd = localPwd
	sess.RemoteFrag = remoteUfrag
	sess.RemotePwd = remotePwd

	return agent, nil
}

func ServeWorkspace(ctx context.Context, dir string) (string, func(context.Context) error, error) {
	caCert, err := makeCert(nil, nil, nil)
	if err != nil {
		return "", nil, err
	}
	serverCert, err := makeCert(
		[]string{"workspace.local"},
		caCert.Certificate[0],
		caCert.PrivateKey,
	)
	if err != nil {
		return "", nil, err
	}

	sess := session{
		TrustCert: string(pem.EncodeToMemory(
			&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Certificate[0]},
		)),
	}
	agent, err := prepareSession(ctx, &sess)
	if err != nil {
		return "", nil, err
	}

	out := new(strings.Builder)
	enc := base64.NewEncoder(base64.StdEncoding, out)
	if err := json.NewEncoder(enc).Encode(&sess); err != nil {
		return "", nil, err
	}
	_ = enc.Close()

	start := func(ctx context.Context) error {
		conn, err := agent.Accept(ctx, sess.RemoteFrag, sess.RemotePwd)
		if err != nil {
			return err
		}

		s := http3.Server{
			Handler: packageServer(dir),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{serverCert},
			},
		}
		go func() {
			<-ctx.Done()
			err := s.CloseGracefully(10 * time.Second)
			if err != nil {
				fmt.Printf("failed to shutdown server: %v", err)
			}
		}()

		return s.Serve(&packetConnAdapter{conn})
	}

	return out.String(), start, nil
}
