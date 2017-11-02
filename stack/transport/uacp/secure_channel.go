package uacp

import (
	"time"

	"github.com/searis/guma/stack/uatype"
)

// Reconnection constants according to OPC UA 1.03 Part 6 section 7.1.6.
const (
	reconnectDelayMultiply = 2
	reconnectMaxDelay      = 2 * time.Minute
)

const (
	sequenceNumberWrapIncr uint32 = 1024
	sequenceNumberWrap     uint32 = 0xFFFFFFFF - 1024
)

// SecureChannel implements the OPC Secure Channel over a raw UACP compatible
// connection.
type SecureChannel struct {
	security ChSecurity
	timeouts Timeouts

	recvWait  chan error
	recvState *recvState
	sendState *sendState

	securityToken uatype.ChannelSecurityToken
}

func newSecureChannel(connMgr *connMgr, security ChSecurity, buffering MsgBuffering, timeouts Timeouts) (*SecureChannel, error) {
	sc := &SecureChannel{
		security: security,
		timeouts: timeouts,
	}

	// Define a monotonic deadline for the secure channel connect operation.
	var deadline time.Time
	if timeouts.ConnectTimeout > 0 {
		deadline = time.Now().Add(timeouts.ConnectTimeout)
	}

	// First connect.
	if err := connMgr.Connect(deadline); err != nil {
		return nil, err
	}

	// On first connect, read chunk settings.
	chunking, _ := connMgr.Chunking()
	sc.sendState = newSendState(connMgr, chunking)
	sc.recvState = newRecvState(connMgr, chunking, buffering)
	sc.recvWait = make(chan error)
	go func() {
		sc.recvWait <- sc.recvState.Run()
	}()

	if err := sc.open(deadline); err != nil {
		logger.LogIfError("newSecureChannel", connMgr.Close())
		return nil, err
	}

	go func() {
		// FIXME: Stop-gap logging only, should handle reconnect.
		if err := <-sc.recvWait; err != nil {
			debugLogger.Println("SecureChannel: recvState.Run closed with error: ", err)
		} else {
			debugLogger.Println("SecureChanne: recvState.Run closed with no error")
		}
	}()
	// TODO: Reconnect UACP and re-open on connection failure.
	// TODO: Re-open on 75% of revised lifetime (after open).
	return sc, nil
}
