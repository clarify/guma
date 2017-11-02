package uacp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/searis/guma/stack/encoding/binary"
	"github.com/searis/guma/stack/transport"
	"github.com/searis/guma/stack/uatype"
)

func (sc *SecureChannel) open(deadline time.Time) error {
	// Wait for receiveQueue spot or deadline.
	requestID, err := sc.recvState.WaitForRequestID(deadline)
	if err != nil {
		return err
	}
	// The receivequeue must always be freed, and it is always safe to cancel,
	// even on success.
	defer sc.recvState.CancelRequestID(requestID)

	// Prepare and encode request.
	var msgBuff bytes.Buffer
	enc := binary.NewEncoder(&msgBuff)

	requestType := uatype.SecurityTokenRequestTypeIssue
	if sc.securityToken.ChannelId != 0 {
		requestType = uatype.SecurityTokenRequestTypeRenew
	}
	var timeoutHint uint32
	if !deadline.IsZero() {
		timeoutHint = encodeUnsignedDuration(time.Until(deadline))
	}
	if err := enc.Encode(uatype.OpenSecureChannelRequest{
		RequestHeader: uatype.RequestHeader{
			Timestamp:   time.Now().UTC(),
			TimeoutHint: timeoutHint,
		},
		RequestType:       requestType,
		SecurityMode:      sc.security.MessageSecurity,
		RequestedLifetime: encodeUnsignedDuration(sc.timeouts.RequestLifetime),
	}); err != nil {
		return transport.LocalError(uatype.StatusBadInternalError, err)
	}

	// Send request.
	if err := sc.sendState.SendMsg(secureMsg{
		Type:           secureMsgTypeOpn,
		ChannelID:      sc.securityToken.ChannelId,
		RequestID:      requestID,
		SecurityHeader: sc.security.SecurityHeader,
		Request: transport.Request{
			NodeID: uatype.NewFourByteNodeID(0, uatype.NodeIdOpenSecureChannelRequest_Encoding_DefaultBinary).Expanded(),
			Body:   &msgBuff,
		},
	}, deadline); err != nil {
		return err
	}

	resp, err := sc.recvState.WaitForResponse(requestID, msgTypeOpn, deadline)
	if err != nil {
		return err
	}
	p, err := ioutil.ReadAll(resp.Body)
	dec := binary.NewDecoder(bytes.NewBuffer(p))
	switch resp.NodeID.Uint() {
	case uatype.NodeIdOpenSecureChannelResponse_Encoding_DefaultBinary:
		target := uatype.OpenSecureChannelResponse{}
		if err := dec.Decode(&target); err != nil {
			return transport.LocalError(uatype.StatusBadInternalError, err)
		}
		if sc.securityToken.ChannelId == 0 {
			sc.securityToken = target.SecurityToken
		}

		// TODO handle more security stuff.
	case uatype.NodeIdServiceFault_Encoding_DefaultBinary:
		target := uatype.ServiceFault{}
		if err := dec.Decode(&target); err != nil {
			return transport.LocalError(uatype.StatusBadInternalError, err)
		}
		return &target
	default:
		err := fmt.Errorf("unexpected node ID %d", resp.NodeID.Uint())
		return transport.LocalError(uatype.StatusBadUnknownResponse, err)
	}

	return nil
}
