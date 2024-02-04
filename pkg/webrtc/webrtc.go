package webrtc

import (
	"fmt"

	"github.com/pion/webrtc/v3"
)

var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

func iceGather(conn *webrtc.PeerConnection) <-chan *webrtc.ICECandidate {
	candidates := make(chan *webrtc.ICECandidate)
	conn.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			fmt.Println("ICE gathering done")
			close(candidates)
			return
		}

		fmt.Printf("got ICE candidate: %v\n", c)
		candidates <- c
	})
	return candidates
}

type webrtcSession struct {
	Offer      webrtc.SessionDescription
	Answer     webrtc.SessionDescription
	Candidates []*webrtc.ICECandidate
}

type Channel interface {
	SetRemoteDescription(webrtc.SessionDescription) error
	AddICECandidate(webrtc.ICECandidateInit) error
}
