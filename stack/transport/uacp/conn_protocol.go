package uacp

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/searis/guma/stack/encoding/binary"
	"github.com/searis/guma/stack/transport"
	"github.com/searis/guma/stack/uatype"
)

const (
	connStateNil uint32 = iota
	connStateConnecting
	connStateConnected
	connStateClosing
	connStateClosed
)

func wrapConnError(err error) error {
	if err == nil {
		return nil
	} else if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		return transport.LocalError(uatype.StatusBadTimeout, nerr)
	} else {
		return transport.LocalError(uatype.StatusBadTcpInternalError, err)
	}
}

// connMgr manages dial of the underlying UACP connection. It is written to
// withstand concurrent access by always resolving to a clean state (as long
// as now internal fields are touched from the outside after the first method
// call).
type connMgr struct {
	// variables used for establishing a connection.
	dial        DialFunc
	endpointURL string
	chunking    MsgChunking

	// variables for accessing the connection.
	connState uint32
	conn      net.Conn

	// if locking both connRecvM and connSendM, do so in alphabetical order to
	// prevent the risk of dead-lock.
	connRecvM sync.Mutex
	connSendM sync.Mutex
}

// newConnMgr returns a connMgr ready for use.
func newConnMgr(dial DialFunc, endpointURL string, chunking MsgChunking) *connMgr {
	return &connMgr{dial: dial, endpointURL: endpointURL, chunking: chunking}
}

// Chunking returns the message chunking agreed during handshake and true if cm
// has been connected at least once. Otherwise a zero value MsgChunking struct
// and false is returned.
func (cm *connMgr) Chunking() (MsgChunking, bool) {
	if atomic.LoadUint32(&cm.connState) < connStateConnected {
		return MsgChunking{}, false
	}
	return cm.chunking, true
}

// Close will close the connetion and move the connMgr state to closed only when
// the original state is connected, otherwise this is a no-op. The operation is
// atomic, and it is therefore safe to call Close multiple times sequentially.
// If Close and Connect is called concurrently, the outcome is arbitrary but
// clean; the connMgr state is always either closed or connected.
func (cm *connMgr) Close() error {
	var err error
	if !atomic.CompareAndSwapUint32(&cm.connState, connStateConnected, connStateClosing) {
		return nil
	}
	err = wrapConnError(cm.conn.Close())
	if !atomic.CompareAndSwapUint32(&cm.connState, connStateClosing, connStateClosed) {
		// It's a coding error if we end up here.
		panic(errors.New("guma/stack/transport/uacp.connMgr.Close: inconsistent connState"))
	}
	return err
}

// Connect will establish an UACP connection, perform the initial handshake and
// move the connMgr state to connected only when the initial state is closed or
// nil (never connected). Otherwise this operation is a no-op. The operation is
// atomic, and it is therefore safe to call Connect multiple times sequentially.
// If Close and Connect is called concurrently, the outcome is arbitrary but
// clean; the connMgr state is always either closed or connected.
func (cm *connMgr) Connect(deadline time.Time) error {
	var firstRun bool
	var nextState = connStateClosed // start with the fallback value to use on errors.

	// Move state to connecting or return.
	if atomic.CompareAndSwapUint32(&cm.connState, connStateNil, connStateConnecting) {
		nextState = connStateNil // fallback value on first run is different.
		firstRun = true
	} else if !atomic.CompareAndSwapUint32(&cm.connState, connStateClosed, connStateConnecting) {
		return nil
	}

	// Make sure to always set a non-transient state on return.
	defer func() {
		if !atomic.CompareAndSwapUint32(&cm.connState, connStateConnecting, nextState) {
			// It's a coding error if we end up here.
			panic(errors.New("guma/stack/transport/uacp.connMgr.Connect: inconsistent connState"))
		}
	}()

	conn, err := cm.dial(deadline)
	if err != nil {
		return transport.LocalError(uatype.StatusBadConnectionRejected, err)
	}

	chunking, err := cm.handshake(conn, deadline)
	if err != nil {
		logger.LogIfError("conn.Close failed during connMgr.Connect", conn.Close())
		return err // Already a ProtocolError.
	} else if !firstRun && !chunking.equals(cm.chunking) {
		// if we are re-establishing a closed connection, the negotiated
		// chunking may not change.
		logger.LogIfError("conn.Close failed during connMgr.Connect", conn.Close())
		return transport.LocalError(uatype.StatusBadConnectionClosed, err)
	}

	// We lock only when we replace conn as connMgr is designed to let
	// SendChunk, RecvChunk fail fast when the connection is closed.
	cm.connRecvM.Lock()
	cm.connSendM.Lock()
	cm.conn = conn
	cm.connRecvM.Unlock()
	cm.connSendM.Unlock()

	nextState = connStateConnected
	return nil
}

func (cm *connMgr) handshake(conn net.Conn, deadline time.Time) (MsgChunking, error) {
	conn.SetWriteDeadline(deadline)
	conn.SetReadDeadline(deadline)

	// buff is reused for send and receive and discared afterwards.
	buff := make([]byte, handshakeMaxSize)

	sendSize := msgHeaderSize + helloMsgSize(cm.endpointURL)
	sendBuff := buff[0:sendSize]
	sendHeader := msgHeader(sendBuff)
	sendHeader.SetHelloHeader(sendSize)

	hello := helloMsg{
		Version:     0,
		MsgChunking: cm.chunking,
		EndpointURL: cm.endpointURL,
	}
	body, err := binary.Marshal(hello)
	if err != nil {
		err = fmt.Errorf("marshal HEL message: %s", err)
		return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpInternalError, err)
	}
	copy(sendBuff[msgHeaderSize:], body)

	debugLogger.Println(">>> Message: HEL")
	debugLogger.Println(sendHeader)
	debugLogger.Spewln(hello)
	debugLogger.Println("=== Message: HEL")

	// Send hello.
	if _, err := conn.Write(sendBuff); err != nil {
		return MsgChunking{}, transport.LocalError(uatype.StatusBadRequestInterrupted, err)
	}

	// Receive ACK or ERR.
	recvBuff := buff
	if _, err := io.ReadAtLeast(conn, recvBuff[:msgHeaderSize], msgHeaderSize); err != nil {
		return MsgChunking{}, transport.LocalError(uatype.StatusBadRequestInterrupted, err)
	}

	recvHeader := msgHeader(recvBuff)
	recvType := recvHeader.Type()
	debugLogger.Println("<<< Message: ", recvType)
	defer debugLogger.Println("=== /Message:", recvType)
	debugLogger.Println(recvHeader)

	size := int(recvHeader.Size())
	switch recvType {
	case msgTypeErr:
		recvBuff = recvBuff[:msgHeaderSize+errMsgMaxSize]
		if size > len(recvBuff) {
			err := errors.New("received ERR message size longer than the maximum allowed value")
			return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpInternalError, err)
		}
		if _, err := io.ReadAtLeast(conn, recvBuff[msgHeaderSize:], size-msgHeaderSize); err != nil {
			return MsgChunking{}, transport.LocalError(uatype.StatusBadRequestInterrupted, err)
		}
		msg := errMsg{}
		if err := binary.Unmarshal(recvBuff[msgHeaderSize:], &msg); err != nil {
			err = fmt.Errorf("unmarshal ERR message: %s", err)
			return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpInternalError, err)
		}
		debugLogger.Spewln(msg)

		return MsgChunking{}, transport.RemoteError(msg.Error, msg.Reason)

	case msgTypeAck:
		recvBuff = recvBuff[:msgHeaderSize+ackMsgSize]
		if size != len(recvBuff) {
			err := errors.New("unexpected ACK message size")
			return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpInternalError, err)
		}
		if _, err := io.ReadAtLeast(conn, recvBuff[msgHeaderSize:], size-msgHeaderSize); err != nil {
			return MsgChunking{}, transport.LocalError(uatype.StatusBadRequestInterrupted, err)
		}
		msg := ackMsg{}
		if err := binary.Unmarshal(recvBuff[msgHeaderSize:], &msg); err != nil {
			err = fmt.Errorf("unmarshal ACK message: %s", err)
			return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpInternalError, err)
		}
		debugLogger.Spewln(msg)

		return msg.MsgChunking, nil

	default:
		err := errors.New("only ACK and ERR messages allowed during handshake")
		// Since conn is closed on error, we don't bother to read out the
		// invalid message from conn. This would have been important if we where
		// planning to keep the connection connected.
		return MsgChunking{}, transport.LocalError(uatype.StatusBadTcpMessageTypeInvalid, err)

	}
}

// SendChunk will write chunk to the underlying connection. If the size is not
// set in the header, this method will set it before sending the chunk. To send
// with no timeout, use a deadline of 0. It is safe to call this method
// concurrently.
func (cm *connMgr) SendChunk(chunk []byte, deadline time.Time) error {
	// Although a single Write to cm.conn is atomic if it's a valid net.Conn
	// implementation, we still need a lock to guard against when Connect
	// replaces cm.conn.
	cm.connSendM.Lock()
	defer cm.connSendM.Unlock()

	h, err := getMsgHeader(chunk)
	if err != nil {
		return err
	}

	// Set msg header size from chunk size, or vise versa.
	if s := int(h.Size()); s == 0 {
		h.SetSize(uint32(len(chunk)))
	} else if s > len(chunk) {
		return errors.New("size in header larger than chunk byte size")
	} else {
		chunk = chunk[:s]
	}

	cm.conn.SetWriteDeadline(deadline)
	_, err = cm.conn.Write(chunk)
	return wrapConnError(err)
}

// RecvChunk will read a chunk from the underlying connection as an atomic
// operation. To send with no timeout, use a deadline of 0. It is safe to call
// this method concurrently.
func (cm *connMgr) RecvChunk(recvBuffer []byte, deadline time.Time) (int, error) {
	cm.connRecvM.Lock()
	defer cm.connRecvM.Unlock()
	var n int

	cm.conn.SetReadDeadline(deadline)

	nn, err := io.ReadAtLeast(cm.conn, recvBuffer[:msgHeaderSize], msgHeaderSize)
	n += nn
	if err != nil {
		return n, wrapConnError(err)
	}

	h := msgHeader(recvBuffer)
	size := h.Size()
	if size > uint32(len(recvBuffer)) {
		return n, transport.LocalError(uatype.StatusBadTcpMessageTooLarge, err)
	}

	nn, err = io.ReadAtLeast(cm.conn, recvBuffer[n:size], int(size)-n)
	n += nn
	if err != nil {
		return n, wrapConnError(err)
	}

	// Validate type of read-out message.
	switch h.Type() {
	case msgTypeOpn, msgTypeClo, msgTypeMsg:
		// Legal secure channel message type, pass along.
		return n, nil
	case msgTypeHel, msgTypeAck, msgTypeRhe:
		// Message type not expected, abort.
		return n, transport.LocalError(
			uatype.StatusBadTcpMessageTypeInvalid,
			errors.New("message type HEL, ACK and RHE only allowed during handshake"),
		)
	case msgTypeErr:
		// An UACP layer error reported, abort.
		msg := errMsg{}
		if err := binary.Unmarshal(recvBuffer[msgHeaderSize:], &msg); err != nil {
			return n, transport.LocalError(
				uatype.StatusBadTcpInternalError,
				fmt.Errorf("during unmarshal of UACP ERR message: %s", err),
			)
		}
		return n, transport.RemoteError(msg.Error, msg.Reason)
	default:
		// Completely foreign message type, abort.
		return n, transport.LocalError(
			uatype.StatusBadTcpMessageTypeInvalid,
			fmt.Errorf("illegal message type %s", h.Type()),
		)
	}
}
