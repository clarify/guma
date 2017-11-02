package uacp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/searis/guma/stack/transport"
	"github.com/searis/guma/stack/uatype"
)

// The SecurityPolicyURIs for supported asymmetric algorithms
const (
	SecurityPolicyURINone           string = "http://opcfoundation.org/UA/SecurityPolicy#None"
	SecurityPolicyURIBasic128Rsa15  string = "http://opcfoundation.org/UA/SecurityPolicy#Basic128Rsa15"
	SecurityPolicyURIBasic256       string = "http://opcfoundation.org/UA/SecurityPolicy#Basic256"
	SecurityPolicyURIBasic256Sha256 string = "http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256"
)

// DefaultValues for Connector.
var (
	DefaultDialTimeout = 30 * time.Second
	DefaultMsgChunking = MsgChunking{
		ReceiveBufferSize: 1024 * 64,
		SendBufferSize:    1024 * 64,
		MaxMessageSize:    1024 * 64 * 1000,
		MaxChunkCount:     1000,
	}
	DefaultMsgBuffering = MsgBuffering{
		RecvBufferCount: 4,
		RecvQueueCount:  24,
	}
	DefaultTimeouts = Timeouts{
		ConnectTimeout:  10 * time.Second,
		RequestLifetime: 1 * time.Hour,
	}
)

// errDeadlineReached should be called when an a deadline occurs outside of
// an UACP send/recv operation.
var errDeadlineReached = transport.LocalError(
	uatype.StatusBadTimeout,
	errors.New("local deadline reached"),
)

// ConnectSecureTCPChannel returns a new SecureChannel over TCP using the
// connection settings defined in opening.
func ConnectSecureTCPChannel(address, endpointURL string, security ChSecurity) (*SecureChannel, error) {
	if err := security.validate(); err != nil {
		return nil, err
	}
	return newSecureChannel(
		newConnMgr(TCPDialFunc(address, DefaultDialTimeout), endpointURL, DefaultMsgChunking),
		security,
		DefaultMsgBuffering,
		DefaultTimeouts,
	)
}

// Connector lets you perform advanced configuration for how to establish a
// Secure Channel over a socket. Multiple SecureChannels may be created by the
// same Connector. All zero values are replaced defaults with an exception of
// ChSecurity and Dial, which is required.
type Connector struct {
	// ChSecurity defines the security settings to request when opening a new
	// channel. This field is required; invalid values or combinations will be
	// rejected.
	ChSecurity ChSecurity

	// Dial is used when connecting or re-connecting the underlying UACP. This
	// field is required; nil values will be panic.
	Dial DialFunc

	// MshChunking is used for the first handshake to negotiate chunk
	// configurations. Could be set to tune performance and memory consuption.
	MsgChunking MsgChunking

	// MsgBuffering defines how many buffers to set up for send and receive
	// operations. Could be set to tune performance and memory consuption.
	MsgBuffering MsgBuffering

	// Timouts is used when connecting or re-connecting a secure channel.
	Timeouts Timeouts
}

// Connect returns a new SecureChannel or an error.
func (c Connector) Connect(endpointURL string) (*SecureChannel, error) {
	if len([]byte(endpointURL)) > msgStingMaxLen {
		return nil, fmt.Errorf("endpoint URL may not be more than %d bytes", msgStingMaxLen)
	}
	if c.Dial == nil {
		panic("c.Dial not set")
	}
	if err := c.ChSecurity.validate(); err != nil {
		return nil, err
	}

	if c.MsgChunking.equals(MsgChunking{}) {
		c.MsgChunking = DefaultMsgChunking
	}
	if c.Timeouts.equals(Timeouts{}) {
		c.Timeouts = DefaultTimeouts
	}
	if c.MsgBuffering.equals(MsgBuffering{}) {
		c.MsgBuffering = DefaultMsgBuffering
	}

	return newSecureChannel(
		newConnMgr(c.Dial, endpointURL, c.MsgChunking),
		c.ChSecurity,
		c.MsgBuffering,
		c.Timeouts,
	)
}

// DialFunc should return a connection that cam be used to transfer UA
// Connection Protocol messages, e.g. a TCP Connection, websocket or other
// connection type that supports full duplex communication. The dial should
// abort when the monotonic time described in deadline is reached.
type DialFunc func(deadline time.Time) (net.Conn, error)

// TCPDialFunc creates a new TCP DialFunc with default settings.
func TCPDialFunc(address string, timeout time.Duration) DialFunc {
	return func(deadline time.Time) (net.Conn, error) {
		d := net.Dialer{Deadline: deadline}
		return d.Dial("tcp", address)
	}
}

// MsgChunking is used to negotiate message chunking configuration as part of
// an OPC UA protocol handshake.
type MsgChunking struct {
	ReceiveBufferSize uint32
	SendBufferSize    uint32
	MaxMessageSize    uint32
	MaxChunkCount     uint32
}

// equals returns true only if all fields in c and other are exactly equal.
func (mc MsgChunking) equals(other MsgChunking) bool {
	return (mc.ReceiveBufferSize == other.ReceiveBufferSize &&
		mc.SendBufferSize == other.SendBufferSize &&
		mc.MaxMessageSize == other.MaxMessageSize &&
		mc.MaxChunkCount == other.MaxChunkCount)
}

// MsgBuffering is used to configure the number of buffers to use for send and
// receive internal to the SecureChannel. Most users should use the default
// settings, but fine-tuning may be relevant in some cases.
type MsgBuffering struct {
	// RecvBufferCount determins how many receive buffers are available.This
	// affects the ability the SecureChannel has to perform pipelineing.
	RecvBufferCount int

	// RecvQueue count determines how many concurrent send-operations that can
	// concurrently wait for a reply, which is mostly relevant for the
	// subscription workflow. Most users should use the default.
	RecvQueueCount int
}

func (mb MsgBuffering) equals(other MsgBuffering) bool {
	return (mb.RecvBufferCount == other.RecvBufferCount &&
		mb.RecvQueueCount == other.RecvQueueCount)
}

// AsymmetricAlgorithmSecurityHeader is used on channel open requests to
// establish a secure connection.
type AsymmetricAlgorithmSecurityHeader struct {
	SecurityPolicyURI             string
	SenderCertificate             uatype.ByteString
	ReceiverCertificateThumbprint uatype.ByteString
}

// equals returns true only if all fields in h and other are exactly equal.
func (sh AsymmetricAlgorithmSecurityHeader) equals(other AsymmetricAlgorithmSecurityHeader) bool {
	return (sh.SecurityPolicyURI == other.SecurityPolicyURI &&
		bytes.Compare(sh.SenderCertificate, other.SenderCertificate) == 0 &&
		bytes.Compare(sh.ReceiverCertificateThumbprint, other.ReceiverCertificateThumbprint) == 0)
}

func (sh AsymmetricAlgorithmSecurityHeader) size() int {
	return 4 + len([]byte(sh.SecurityPolicyURI)) + (sh.SenderCertificate.BitLength()+sh.ReceiverCertificateThumbprint.BitLength())/8
}

// ChSecurity defines the security settings for a secure channel.
type ChSecurity struct {
	SecurityHeader  AsymmetricAlgorithmSecurityHeader
	MessageSecurity uatype.MessageSecurityMode
}

// equals returns true only if all fields in h and other are exactly equal.
func (cs ChSecurity) equals(other ChSecurity) bool {
	return (cs.SecurityHeader.equals(other.SecurityHeader) &&
		cs.MessageSecurity == other.MessageSecurity)
}

func (cs ChSecurity) validate() error {
	switch cs.MessageSecurity {
	case uatype.MessageSecurityModeNone:
		if cs.SecurityHeader.SecurityPolicyURI == SecurityPolicyURINone {
			return nil
		}
		return transport.LocalError(uatype.StatusBadSecurityPolicyRejected, nil)
	case uatype.MessageSecurityModeSign:
		// TODO: implement support for signing!
		return transport.LocalError(uatype.StatusBadNotImplemented, nil)
	case uatype.MessageSecurityModeSignAndEncrypt:
		// TODO: implement support for encryption and signing!
		return transport.LocalError(uatype.StatusBadNotImplemented, nil)
	default:
		return transport.LocalError(uatype.StatusBadSecurityPolicyRejected, nil)
	}
}

// Timeouts defines values related to setting of deadlines.
type Timeouts struct {
	// ConnectTimeout is how long an entire connection process (UACP creation,
	// handshake and secure channel creation) may last before giving up.
	ConnectTimeout time.Duration

	// RequestLifetime is how long a client request a secure channel to stay
	// open. The server might respond with a revised lifetime. When 75% of
	// the revised lifetime has passed, the client wil renew the secure channel
	// with the same request lifetime.
	RequestLifetime time.Duration
}

// equals returns true only if all fields in h and other are exactly equal.
func (ts Timeouts) equals(other Timeouts) bool {
	return (ts.ConnectTimeout == other.ConnectTimeout &&
		ts.RequestLifetime == other.RequestLifetime)
}
