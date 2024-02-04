package webrtc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pion/webrtc/v3"
)

func RemoteChannel(ctx context.Context, localSession string, dst io.Writer) error {
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}
	defer func() {
		_ = peerConnection.Close()
	}()

	iceGatheringDone := iceGather(peerConnection)
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())
	})
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			_, err := dst.Write(msg.Data)
			if err != nil {
				fmt.Printf("failed to write to destination: %v\n", err)
				cancel()
			}
		})
		d.OnClose(cancel)
	})

	sessJSON, err := base64.StdEncoding.DecodeString(localSession)
	if err != nil {
		return err
	}
	var sess webrtcSession
	if err := json.Unmarshal(sessJSON, &sess); err != nil {
		return err
	}

	if err := peerConnection.SetRemoteDescription(sess.Offer); err != nil {
		return err
	}

	for _, candidate := range sess.Candidates {
		if err := peerConnection.AddICECandidate(candidate.ToJSON()); err != nil {
			return err
		}
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := peerConnection.SetLocalDescription(answer); err != nil {
		return err
	}

	fmt.Printf("WEBRTC-ANSWER:%s\n", answer.SDP)
	for candidate := range iceGatheringDone {
		fmt.Printf("WEBRTC-CANDIDATE:%s\n", candidate.String())
	}

	<-ctx.Done()
	return nil
}
