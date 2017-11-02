package uacp

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/searis/guma/stack/transport"
	"github.com/searis/guma/stack/uatype"

	"github.com/searis/guma/stack/encoding/binary"
)

// maxRecvQueues is the upper limit of receive buffers allowed by the
// requestIDQueueIndexMask. If a higher value that this is asked for during
// initialization, a panic wil be raised.
const maxRecvQueues = 0x7FFF

const (
	requestIDQueueIndexMask uint32 = 0x00007FFF
	requestIDInvalidMask    uint32 = 0x00008000
	requestIDMonotonicMask  uint32 = 0xFFFF0000
	requestIDMonotonicIncr  uint32 = 0x00010000
)

const recvEventRouteTimeout time.Duration = 5 * time.Second

// recvEvent represents a notification about a new message chunk.
type recvEvent struct {
	// If err is set, msgHeader and body should be ignored.
	err error

	msgHeader secureMsgHeader

	// body contains a slice of the receive buffer pointing to the message body
	// or nil on err.
	body []byte

	// freeBuffer must always be called to put the receive buffer back in the
	// queue, also on errors. After the first call, all later calls will be a
	// no-op.
	freeBuffer func()
}

// A recvQueue allows routing chunks from the receive go-routine to a waiting
// operation.
type recvQueue struct {
	// requestID is used to indicate who is listening on events (if anyone),
	// as well as to ensure that the recvQueue spot gets freed exactly once.
	requestID uint32

	// isOpen is used to prevent the receive thread from sending any messages if
	// events channel have been closed. To ensure this, it is atomically set to
	// 1 only be the listener, and to 0 only by the receive thread.
	isOpen uint32

	// chunks is an a buffered channel used for delivery of message chunks or
	// errors. All events must eventually be handled by calling the freeBuffer
	// method on them.
	events chan recvEvent
}

// recvState handles the receiver thread that continuously reads from connMgr.
// It also manages sender request ID generation, and association of receive
// queues to these request IDs that can be used to listen for server
// responses. The sender must always free
//
//  TODO: Should also handle signature checks and decryption.
type recvState struct {
	connMgr *connMgr

	// Message size limitations.
	maxMessageSize uint32
	maxChunkCount  uint32

	securityTokenM sync.Mutex
	securityToken  uatype.ChannelSecurityToken

	// monotonicRequestID is used to generate a sender's requestIDs. The
	// monotonic part is monotonic based on recvQueue allocation time, not based
	// on sending order.
	monotonicRequestID uint32

	// lastRequestID and lastChunkNo is used to check that messages and chunks
	// arrive in the expected orders and guard against replay attacks.
	lastRequestID uint32
	lastChunkNo   uint32

	// buffers holds a pool of free receive buffers.
	buffers chan []byte

	// queues are used to send a full receivebuffer to a waiting operation.
	queues []recvQueue
	// queueSpots are used to allocate an index in queues. The allocated index
	// is used as part of send request IDs so that it's easy to route incoming
	// messages to the right receiver.
	queueSpots chan int
}

func newRecvState(connMgr *connMgr, chunking MsgChunking, buffering MsgBuffering) *recvState {
	if buffering.RecvBufferCount > maxRecvQueues {
		panic(fmt.Errorf("don't allow more than %d receive queues", maxRecvQueues))
	}
	if buffering.RecvBufferCount == 0 {
		panic(errors.New("recv buffer count may not be 0"))
	}
	if buffering.RecvQueueCount == 0 {
		panic(errors.New("recv queue count may not be 0"))
	}
	rcv := recvState{
		connMgr:        connMgr,
		maxMessageSize: chunking.MaxMessageSize,
		maxChunkCount:  chunking.MaxChunkCount,

		lastRequestID: requestIDInvalidMask,

		buffers:    make(chan []byte, buffering.RecvBufferCount),
		queues:     make([]recvQueue, buffering.RecvQueueCount),
		queueSpots: make(chan int, buffering.RecvQueueCount),
	}
	for i := 0; i < cap(rcv.buffers); i++ {
		rcv.buffers <- make([]byte, chunking.ReceiveBufferSize)
	}
	for i := 0; i < len(rcv.queues); i++ {
		rcv.queueSpots <- i
		rcv.queues[i].requestID = requestIDInvalidMask
	}
	return &rcv
}

// WaitForRequestID will wait for a receive queue to be free or the deadline to
// be reached. On success, it assigns the queue and returns a Request ID that
// should be used when sending request messages, and for waiting for a response.
// either WaitForResponse ro CancelRequestID must be called to free the
// associated receive queue. Failing to do so may result in leaks!
func (rcv *recvState) WaitForRequestID(deadline time.Time) (uint32, error) {
	var timeout <-chan time.Time
	if deadline.IsZero() {
		timeout = time.After(time.Until(deadline))
	}
	select {
	case qi := <-rcv.queueSpots:
		requestID := uint32(qi) | atomic.AddUint32(&rcv.monotonicRequestID, requestIDMonotonicIncr)
		atomic.StoreUint32(&rcv.queues[qi].requestID, requestID)
		rcv.queues[qi].events = make(chan recvEvent, len(rcv.buffers))
		if !atomic.CompareAndSwapUint32(&rcv.queues[qi].isOpen, 0, 1) {
			// If we end-up here, we have a serious internal programming error.
			// The index for an open queue should never be in the queuesSpots
			// channel!
			panic(errors.New("guma/stack/transport/uacp.recvState.WaitForRequestID: bad queue state"))
		}
		return requestID, nil
	case <-timeout:
		return 0, errDeadlineReached
	}

}

// Wait for response will wait for all message chunks to be received, or the
// deadline to be reached. Incoming chunks are validated against t as well as
// internal rcv values. When all chunks have been received, or if the deadline
// is reached, CancelRequestID is called to free the receive queue.
func (rcv *recvState) WaitForResponse(requestID uint32, t msgType, deadline time.Time) (*transport.Response, error) {
	defer rcv.CancelRequestID(requestID)

	debugLogger.Printf(
		"recvState.WaitForResponse: waiting for request ID %d\n",
		requestID,
	)

	queue, err := rcv.eventQueue(requestID)
	if err != nil {
		return nil, transport.LocalError(uatype.StatusBadInternalError, err)
	}
	timeout := time.After(time.Until(deadline))

	var chunkCnt uint32
	var nodeID uatype.ExpandedNodeId
	var buff *bytes.Buffer

	// Read first chunk, and get NodeID
	select {
	case e, ok := <-queue:
		if !ok {
			return &transport.Response{NodeID: nodeID, Body: buff}, nil
		}
		chunkCnt++

		if err := rcv.validateChunkMsgHeaders(e.msgHeader, t); err != nil {
			logger.LogIfError("connMgr.Close during recvState.WaitForResponse", rcv.connMgr.Close())
			return nil, err
		}
		if e.err != nil {
			e.freeBuffer()
			return nil, e.err
		}

		// Hack to decode first chunk with Node ID

		err := binary.Unmarshal(e.body, &nodeID)
		e.freeBuffer()
		if err != nil {
			return nil, transport.LocalError(
				uatype.StatusBadUnknownResponse,
				errors.New("could not decode NodeID: "+err.Error()),
			)
		}
		buff = bytes.NewBuffer(e.body[nodeID.Size():])
	case <-timeout:
		return nil, errDeadlineReached
	}

	for {
		select {
		case e, ok := <-queue:
			if !ok {
				return &transport.Response{NodeID: nodeID, Body: buff}, nil
			}
			chunkCnt++

			if err := rcv.validateChunkMsgHeaders(e.msgHeader, t); err != nil {
				logger.LogIfError("connMgr.Close during recvState.WaitForResponse", rcv.connMgr.Close())
				return nil, err
			}
			if e.err != nil {
				e.freeBuffer()
				return nil, e.err
			}
			if _, err := buff.Write(e.body); err != nil {
				return nil, transport.LocalError(uatype.StatusBadInternalError, err)
			}

			// Are we done yet?
			e.freeBuffer()
		case <-timeout:
			return nil, errDeadlineReached
		}
		if chunkCnt >= rcv.maxChunkCount {
			err := fmt.Errorf("counter more than %d chunks", rcv.maxChunkCount)
			return nil, transport.LocalError(uatype.StatusBadResponseTooLarge, err)

		}
	}
}

func (rcv *recvState) SetSecurityToken(st uatype.ChannelSecurityToken) {
	rcv.securityTokenM.Lock()
	rcv.securityToken = st
	rcv.securityTokenM.Unlock()

}

func (rcv *recvState) validateChunkMsgHeaders(h secureMsgHeader, t msgType) error {
	rcv.securityTokenM.Lock()
	defer rcv.securityTokenM.Unlock()

	if h.msgType() != t {
		// Unexpected message type.
		return transport.LocalError(
			uatype.StatusBadTcpMessageTypeInvalid,
			fmt.Errorf("unexpected message type %s in response", h.msgType()),
		)
	}
	if id := rcv.securityToken.ChannelId; id != 0 && id != h.SecureChannelID {
		return transport.LocalError(
			uatype.StatusBadTcpSecureChannelUnknown,
			fmt.Errorf("unexpected secure channel ID %d in response", id),
		)
	}
	return nil
}

// CancelRequestID will fire off a backround process which will free the receive
// queue associated with the requestID, as well as any receive buffers assigned
// to recvEvents in the queue. If this function is called multiple times with
// the same request ID, only the first call will take effect.
func (rcv *recvState) CancelRequestID(requestID uint32) {
	qi := int(requestID & requestIDQueueIndexMask)

	// Free the queue spot only once.
	if qi < len(rcv.queues) && atomic.CompareAndSwapUint32(&rcv.queues[qi].requestID, requestID, requestID|requestIDInvalidMask) {
		// Fire of a clean-up routine.
		go func() {
			// empty channel before delivering back the queue spot.
			for e := range rcv.queues[qi].events {
				e.freeBuffer()
			}
			if atomic.LoadUint32(&rcv.queues[qi].isOpen) != 0 {
				// If we end-up here, we have a serious internal programming error.
				// Once the channel is closed, and before the index is placed back in queueSpots, isOpen should always be 0.
				panic(errors.New("guma/stack/transport/uacp.recvState.FreeRequestID: bad queue state"))
			}
			rcv.queueSpots <- qi
		}()

	}
}

func (rcv *recvState) eventQueue(requestID uint32) (<-chan recvEvent, error) {
	qi := int(requestID & requestIDQueueIndexMask)
	if qi >= len(rcv.queues) {
		return nil, errors.New("invalid request ID FUUU")
	}
	return rcv.queues[qi].events, nil
}

// Run will run until an error that require reconnection of the UACP occurs. On
// final close of a secure channel, the returned error may be ignored.
func (rcv *recvState) Run() error {
	for {
		e, requestID := rcv.waitForEvent()
		if requestID&requestIDInvalidMask != 0 {
			e.freeBuffer()
			logger.Println("recvState: closing connMgr due to error:", e.err)
			logger.LogIfError("recvState.Run: recvState.ConnMgr.Close:", rcv.connMgr.Close())
			return e.err
		}
		rcv.routeEvent(requestID, e)

	}
}

// waitForEvent receives exactly one message chunk, validates the sequence
// header, and returns a recvEvent and a requestID. The recvEvent may contain an
// error, and the requestID may be invalid.
//
// TODO: Handle message security (e.g. encryption and signature) here?
func (rcv *recvState) waitForEvent() (recvEvent, uint32) {
	var i int
	var seqh sequenceHeader
	var buffFreed uint32

	// Wait for free receive buffer
	buff := <-rcv.buffers
	e := recvEvent{
		freeBuffer: func() {
			if atomic.CompareAndSwapUint32(&buffFreed, 0, 1) {
				rcv.buffers <- buff
			}
		},
	}

	// Receive message header with no timeout; will error if the connection
	// gets closed.
	msgSize, err := rcv.connMgr.RecvChunk(buff, time.Time{})
	if err != nil {
		e.err = err
		return e, requestIDInvalidMask
	}

	// Decode msg header.
	if err := binary.Unmarshal(buff[0:secureMsgHeaderSize], &e.msgHeader); err != nil {
		e.err = err
		return e, requestIDInvalidMask
	}
	i += secureMsgHeaderSize

	// Decode security header.
	switch e.msgHeader.msgType() {
	case msgTypeOpn, msgTypeClo:
		ah := AsymmetricAlgorithmSecurityHeader{}
		if err := binary.Unmarshal(buff[i:], &ah); err != nil {
			e.err = transport.LocalError(uatype.StatusBadInternalError, err)
			return e, requestIDInvalidMask
		}
		i += ah.size()
	case msgTypeMsg:
		sh := symmetricAlgorithmSecurityHeader{}
		if err := binary.Unmarshal(buff[i:], &sh); err != nil {
			e.err = err
			return e, requestIDInvalidMask
		}
		i += symmetricAlgorithmSecurityHeaderSize
		// TODO: set flags to check message signature and/or encryption here?
	default:
		// this should never happen, as invalid/unknown message types should
		// have been filtered by the connMgr.
		panic(errors.New("guma/stack/transport/uacp.recvState.RecvChunk: invalid message type"))
	}

	// Decode and validate sequence headers.
	if err := binary.Unmarshal(buff[i:], &seqh); err != nil {
		e.err = err
		return e, requestIDInvalidMask
	}
	i += sequenceHeaderSize

	if seqh.SequenceNumber != rcv.lastChunkNo+1 &&
		(e.msgHeader.msgType() != msgTypeOpn || rcv.lastChunkNo <= sequenceNumberWrap) {
		err := fmt.Errorf("got sequence number %d, expected %d", seqh.SequenceNumber, rcv.lastChunkNo+1)
		e.err = transport.LocalError(uatype.StatusBadSequenceNumberInvalid, err)
		return e, requestIDInvalidMask
	}
	rcv.lastChunkNo = seqh.SequenceNumber

	if rcv.lastRequestID&requestIDInvalidMask == 0 && seqh.RequestID != rcv.lastRequestID {
		err = errors.New("got new request ID after an intermediate chunk")
		e.err = transport.LocalError(uatype.StatusBadUnknownResponse, err)
		return e, requestIDInvalidMask
	}
	switch e.msgHeader.ChunkType {
	case chunkTypeFinal, chunkTypeFinalAborted:
		rcv.lastRequestID = seqh.RequestID | requestIDInvalidMask
	case chunkTypeIntermediate:
		rcv.lastRequestID = seqh.RequestID
	}

	// Retrieve message body or error and return for routing.
	// FIXME: handle signature and encryption before setting e.body / e.err
	switch e.msgHeader.ChunkType {
	case chunkTypeFinalAborted:
		var abort secureAbortBody
		if err := binary.Unmarshal(buff[i:msgSize], &abort); err != nil {
			e.err = transport.LocalError(uatype.StatusBadInternalError, err)
		} else {
			e.err = transport.RemoteError(abort.Status, abort.Reason)
		}
	case chunkTypeIntermediate, chunkTypeFinal:
		e.body = buff[i:msgSize]
	}
	return e, seqh.RequestID

}

// routeEvent will attempt to send e on the right queue, and close the event
// channel on final chunks or timeouts.
func (rcv *recvState) routeEvent(requestID uint32, e recvEvent) {
	qi := int(requestID & requestIDQueueIndexMask)
	ch := rcv.queues[qi].events
	debugLogger.Printf(
		"recvState: routing chunk for request ID %d to queue %d, header: %#v\n",
		requestID, qi, e.msgHeader,
	)

	// Check if the chunk should be discarded.
	if atomic.LoadUint32(&rcv.queues[qi].isOpen) == 0 {
		// We have closed the channel for this queue previously, possibly due
		// to a timeout.
		debugLogger.Printf(
			"recvState: discarding chunk for request ID %d due to closed queue\n",
			requestID,
		)
		e.freeBuffer()
		return
	} else if id := atomic.LoadUint32(&rcv.queues[qi].requestID); id != requestID {
		// Most likely we have closed the channel for this queue previously,
		// possibly due to a timeout. The queue have been reassigned to a new
		// request ID, but we ar still receiving mesage chunks from the previous
		// request ID assigned to this spot.
		debugLogger.Printf(
			"recvState: discarding chunk for request ID %d due to no listener\n",
			requestID,
		)
		e.freeBuffer()
		return
	}

	// Try to send the chunk to the waiting listener through a buffered channel,
	// or timeout and close the channel.
	select {
	case ch <- e:
		switch e.msgHeader.ChunkType {
		case chunkTypeFinalAborted, chunkTypeFinal:
			goto CLOSE
		}
	case <-time.After(recvEventRouteTimeout):
		e.freeBuffer()
		debugLogger.Printf(
			"recvState: gave up routing a chunk for request ID %d after %s\n",
			requestID, recvEventRouteTimeout,
		)
		goto CLOSE
	}

	return

CLOSE:
	if !atomic.CompareAndSwapUint32(&rcv.queues[qi].isOpen, 1, 0) {
		// If we end-up here, we have a serious internal programming error.
		// Nobody else except us should be closing the event channel.
		panic(errors.New("guma/stack/transport/uacp.recvState.routeEvent: bad queue state"))
	}
	close(ch)
}

// closeQueue close the queue at index qi if it's open.
func (rcv *recvState) closeQueue(qi int) {

}
