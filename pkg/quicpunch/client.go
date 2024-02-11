package quicpunch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pion/ice/v2"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func FetchWorkspace(ctx context.Context, peerInfo string, dst io.Writer) error {
	peerInfoJSON, err := base64.StdEncoding.DecodeString(peerInfo)
	if err != nil {
		return err
	}
	var sess session
	if err := json.Unmarshal(peerInfoJSON, &sess); err != nil {
		return err
	}

	agent, err := startAgent(ctx, sess.RemoteFrag, sess.RemotePwd, false)
	if err != nil {
		return err
	}
	if err := agent.SetRemoteCredentials(sess.LocalFrag, sess.LocalPwd); err != nil {
		return err
	}
	for _, candidateRaw := range sess.Candidates {
		candidate, err := ice.UnmarshalCandidate(candidateRaw)
		if err != nil {
			return err
		}
		if err := agent.AddRemoteCandidate(candidate); err != nil {
			return err
		}
	}

	// if err := agent.GatherCandidates(); err != nil {
	// 	return err
	// }

	conn, err := agent.Dial(ctx, sess.LocalFrag, sess.LocalPwd)
	if err != nil {
		return err
	}

	tr := quic.Transport{
		Conn: &packetConnAdapter{conn},
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM([]byte(sess.TrustCert))
	client := http.Client{
		Transport: &http3.RoundTripper{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
			Dial: func(ctx context.Context, addr string, tlsConf *tls.Config, quicConf *quic.Config) (quic.EarlyConnection, error) {
				return tr.DialEarly(ctx, conn.RemoteAddr(), tlsConf, quicConf)
			},
		},
	}

	resp, err := client.Get("https://workspace.local/")
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code response: %d", resp.StatusCode)
	}

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return err
	}

	return nil
}
