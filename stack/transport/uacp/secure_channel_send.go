package uacp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/searis/guma/stack/encoding/binary"
	"github.com/searis/guma/stack/uatype"

	"github.com/searis/guma/stack/transport"
)

// Send sends a request through an open channel or times out. To run with no
// timeout, let deadaline be the zero time.
func (sc *SecureChannel) Send(r transport.Request, deadline time.Time) (*transport.Response, error) {
	// Wait for receiveQueue spot or deadline.
	requestID, err := sc.recvState.WaitForRequestID(deadline)
	if err != nil {
		return nil, err
	}
	// The receivequeue must always be freed, and it is always safe to cancel,
	// even on success.
	defer sc.recvState.CancelRequestID(requestID)

	if err != nil {
		return nil, transport.LocalError(uatype.StatusBadInternalError, err)
	}

	if err := sc.sendState.SendMsg(secureMsg{
		Type:           secureMsgTypeMsg,
		ChannelID:      sc.securityToken.ChannelId,
		RequestID:      requestID,
		SecurityHeader: symmetricAlgorithmSecurityHeader{TokenID: sc.securityToken.TokenId},
		Request:        r,
	}, deadline); err != nil {
		return nil, err
	}

	return sc.recvState.WaitForResponse(requestID, msgTypeMsg, deadline)
}

// sendState manages chunking and sending of messages. In the future it will
// also need to handle encryption and signing.
type sendState struct {
	sync.Mutex
	sendBuffer []byte
	connMgr    *connMgr

	// lastChunkNo is a monotonically increased chunk count, that is only
	// wrapped on secure channel lifetime renewal requests when chunkNo >
	// MaxUint32 - 1024. The wrap is implemented by adding 1024 while relying on
	// overflow.
	lastChunkNo uint32

	// Message size limitations.
	maxMessageSize uint32
	maxChunkCount  uint32
}

func newSendState(connMgr *connMgr, chunking MsgChunking) *sendState {
	return &sendState{
		connMgr:        connMgr,
		sendBuffer:     make([]byte, chunking.SendBufferSize),
		maxMessageSize: chunking.MaxMessageSize,
		maxChunkCount:  chunking.MaxChunkCount,
	}
}

// secureMsg is used only to rely information to the sendState.SendMsg, and
// does not describe the message encoding order.
type secureMsg struct {
	Type           [3]byte
	ChannelID      uint32
	RequestID      uint32
	SecurityHeader interface{}
	Request        transport.Request
}

// SendMsg will send msg as one or more chunks through the underlying UACP
// connection. Sequence headers are automatically added.
func (snd *sendState) SendMsg(msg secureMsg, deadline time.Time) error {
	// Valdidate securityHeader type.
	switch msg.SecurityHeader.(type) {
	case AsymmetricAlgorithmSecurityHeader, symmetricAlgorithmSecurityHeader:
	default:
		panic(errors.New("invalid security header type"))
	}

	snd.Lock()
	defer snd.Unlock()

	// Encode NodeID. The most common NodeID size is 4 bytes, so we optimize
	// for that.
	buf := bytes.NewBuffer(make([]byte, 0, 4))
	enc := binary.NewEncoder(buf)
	if err := enc.Encode(msg.Request.NodeID); err != nil {
		err = fmt.Errorf("while encoding NodeID: %s", err)
		return transport.LocalError(uatype.StatusBadInternalError, err)
	}

	// msgReader will let us read the NodeID and request Body in sequence.
	msgReader := io.MultiReader(buf, msg.Request.Body)

	// Determine first chunk number.
	var firstChunkNo uint32
	if msg.Type == secureMsgTypeOpn && snd.lastChunkNo >= sequenceNumberWrap {
		// Perform a controlled overflow of the sequence number, which is
		// only allowed on channel renewals when chunkNo > sequenceNumberWrap.
		firstChunkNo = sequenceNumberWrapIncr
	} else {
		firstChunkNo = snd.lastChunkNo + 1
	}
	chunkNo := firstChunkNo

	// Prepare and send chunks.
	moreChunks := true
	for moreChunks {
		buf := newFixedSizeBuffer(snd.sendBuffer)
		enc := binary.NewEncoder(buf)

		// Write all headers. Note that we can't send an abort message if
		// encoding the message headers fails, so in that case we will just
		// return.
		if err := enc.Encode(secureMsgHeader{
			Type:      msg.Type,
			ChunkType: chunkTypeFinal, // start out with Final, change it if needed.
		}); err != nil {
			err = fmt.Errorf("sequence number %d: message header encode: %s", chunkNo, err)
			return transport.LocalError(uatype.StatusBadInternalError, err)
		}
		if err := enc.Encode(msg.SecurityHeader); err != nil {
			err = fmt.Errorf("sequence number %d: security header encode: %s", chunkNo, err)
			return transport.LocalError(uatype.StatusBadInternalError, err)
		}
		if err := enc.Encode(sequenceHeader{
			RequestID:      msg.RequestID,
			SequenceNumber: chunkNo,
		}); err != nil {
			err = fmt.Errorf("sequence number %d: sequence header encode: %s", chunkNo, err)
			return transport.LocalError(uatype.StatusBadInternalError, err)
		}
		headerSize := enc.BytesWritten()
		maxBodySize := int64(buf.Cap()) - headerSize

		// Write to chunk from body
		var terr *transport.Error
		moreChunks = false

		if n, err := io.CopyN(buf, msgReader, maxBodySize); n > 0 && err == io.EOF {
			// Everything is OK, and there are no more chunks.
		} else if err != nil {
			// Error; set terr so we can send an abort chunk if needed.
			switch t := err.(type) {
			case *transport.Error:
				terr = t
			default:
				terr = transport.LocalError(uatype.StatusBadInternalError, err)
			}
		} else if n == maxBodySize {
			// Test if there are more chunks, and if we are allowed to send more
			// chunks. On errors, set terr to signal abortion.
			carry := newFixedSizeBuffer(make([]byte, 16))
			if cn, cerr := io.CopyN(carry, msgReader, 16); cerr == io.EOF && cn > 0 {
				// more chunks, all the data fit in the carry buffer.
				setChunkType(buf.Bytes(), chunkTypeIntermediate)
				msgReader = carry
				moreChunks = true
			} else if cerr == nil && cn > 0 {
				// More chunks, all data may not have fit in the carry-buffer.
				setChunkType(buf.Bytes(), chunkTypeIntermediate)
				msgReader = io.MultiReader(carry, msgReader)
				moreChunks = true
			} else if cerr != nil {
				// error; set terr so we can send an abort chunk if needed.
				terr = transport.LocalError(uatype.StatusBadInternalError, cerr)
			} else if cn > 0 && chunkNo-firstChunkNo == snd.maxChunkCount {
				reason := fmt.Errorf("maximum number of chunks (%d) reached", snd.maxChunkCount)
				terr = transport.LocalError(uatype.StatusBadTcpMessageTooLarge, reason)
			}
		}

		// In case of an error, determine if we need to send an abort message.
		if terr != nil {
			// If we are on the first chunk, no need to send an abort message.
			if firstChunkNo == chunkNo {
				return terr
			}
			// Rewrite buf to contian an abort message.
			buf.Truncate(int(headerSize))
			setChunkType(buf.Bytes(), chunkTypeFinalAborted)
			if err := enc.Encode(secureAbortBody{
				Status: terr.StatusCode(),
				Reason: terr.Reason().Error(),
			}); err != nil {
				err = errors.New("failed while encoding abort mesage: " + err.Error())
				return transport.LocalError(uatype.StatusBadInternalError, err)
			}
		}

		if err := snd.connMgr.SendChunk(buf.Bytes(), deadline); err != nil {
			return err
		}
		snd.lastChunkNo = chunkNo
		chunkNo++

		if terr != nil {
			return terr
		}
	}
	return nil
}
