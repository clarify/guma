package transport

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/searis/guma/stack/encoding/binary"
	"github.com/searis/guma/stack/uatype"
)

// Default values for SecureChannel
const (
	DefaulSecurityPolicyURI string        = SecurityPolicyURINone
	DefaultLifeTime         time.Duration = 12 * time.Hour
)

// SecureChannel implements the OPC Secure Channel. This struct will use default values if no
// user params are provided.
type SecureChannel struct {
	c                             Conn
	SecurityPolicyURI             string
	SenderCertificate             uatype.ByteString
	ReceiverCertificateThumbPrint uatype.ByteString
	LifeTime                      time.Duration
	securityTokenID               uint32
	secureChannelID               uint32
	sequenceHeader                sequenceHeader
	errorChan                     chan (error)
	renewTimer                    *time.Timer
	//Todo: add SecurityMode uatype.enumMessageSecurityMode
}

// OpenSecureChannel opens a SecureChannel with default values and returns a channel ready to use.
// The provided connection must be opened and valid before calling this method.
// It is the callers responsibility to close the provided Conn after closing the SecureChannel.
func OpenSecureChannel(c Conn, errChan chan (error)) (*SecureChannel, error) {
	secChan := &SecureChannel{}
	err := secChan.Open(c, errChan)
	return secChan, err
}

// Open opens a SecureChannel with a provided connection.
// The provided connection must be opened and valid before calling this method.
// It is the callers responsibility to close the provided Conn after closing the SecureChannel.
// The SecureChannel will keep the Channel open by renewing the channel on 70% elapsed lifetime. Errors during renewal
// will be provided in the errChan.
func (s *SecureChannel) Open(c Conn, errChan chan (error)) error {
	if c == nil {
		return fmt.Errorf("nil conn provided")
	}
	s.c = c
	s.errorChan = errChan
	msg := openSecureChannelRequestMsg{
		SecurityHeader: asymmetricAlgorithmSecurityHeaderNone(),
		SequenceHeader: s.nextSequenceHeader(),
		TypeID:         uatype.NewFourByteNodeID(0, uatype.NodeIdOpenSecureChannelRequest_Encoding_DefaultBinary),
		SecureChannelRequest: uatype.OpenSecureChannelRequest{
			RequestHeader: uatype.RequestHeader{
				Timestamp: time.Now(),
			},
			RequestType:       uatype.SecurityTokenRequestTypeIssue,
			SecurityMode:      uatype.MessageSecurityModeNone,
			RequestedLifetime: s.lifeTime(),
		},
	}
	data, err := msg.bytes()
	if err != nil {
		return err
	}
	err = c.Send(MessageTypeOpen, bytes.NewReader(data), 0)
	if err != nil {
		return err
	}

	fmt.Println("Lets receive")
	rdr, err := c.Receive()
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(rdr)
	if err != nil {
		return err
	}

	resp := openSecureChannelResponseMsg{}
	err = resp.decode(data)

	s.secureChannelID = resp.SecureChannelResponse.SecurityToken.ChannelId
	s.LifeTime = time.Duration(resp.SecureChannelResponse.SecurityToken.RevisedLifetime)
	s.securityTokenID = resp.SecureChannelResponse.SecurityToken.TokenId

	//TODO: Make renew function
	s.renewTimer = time.AfterFunc(time.Duration(s.lifeTime()), func() { fmt.Println("Lets renew") })

	return nil
}

// Send transmits a message over the secureChannel to the connected endpoint and blocks until a valid
// response is received.
func (s *SecureChannel) Send(r *Request) (*Response, error) {
	//TODO: MUtex here

	msg := secureChannelMsg{
		SecurityHeader: symmetricAlgorithmSecurityHeader{
			TokenID: s.securityTokenID,
		},
		SequenceHeader: s.nextSequenceHeader(),
		TypeID:         r.NodeID,
		body:           r.Body,
	}
	data, err := msg.bytes()
	if err != nil {
		return nil, err
	}

	if err := s.c.Send(MessageTypeMsg, bytes.NewReader(data), s.secureChannelID); err != nil {
		return nil, err
	}

	resp, err := s.c.Receive()
	if err != nil {
		return nil, err
	}

	var headBuf bytes.Buffer
	if _, err := io.CopyN(&headBuf, resp, 16); err != nil {
		return nil, err
	}

	msg = secureChannelMsg{}
	if err := msg.decodeHeader(&headBuf); err != nil {
		return nil, err
	}

	// TODO: validate some params from secChanMsg

	return &Response{
		NodeID: msg.TypeID,
		Body:   resp,
	}, nil
}

// Close initiates a CloseSecureChannel request and returns on completion.
// Note: The underlying Conn interface still needs to be closed by the user.
func (s *SecureChannel) Close() error {
	//TODO: send CloseSecureChannelMsg
	return nil
}

func (s *SecureChannel) nextSequenceHeader() sequenceHeader {
	s.sequenceHeader.SequenceNumber++
	s.sequenceHeader.RequestID++
	return s.sequenceHeader
}

type openSecureChannelRequestMsg struct {
	SecurityHeader       asymmetricAlgorithmSecurityHeader
	SequenceHeader       sequenceHeader
	TypeID               uatype.NodeId
	SecureChannelRequest uatype.OpenSecureChannelRequest
}

func (o openSecureChannelRequestMsg) bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)
	err := enc.Encode(o)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type openSecureChannelResponseMsg struct {
	SecurityHeader        asymmetricAlgorithmSecurityHeader
	SequenceHeader        sequenceHeader
	TypeID                uatype.NodeId
	SecureChannelResponse uatype.OpenSecureChannelResponse
}

func (o *openSecureChannelResponseMsg) decode(data []byte) error {
	dec := binary.NewDecoder(bytes.NewReader(data))
	return dec.Decode(o)
}

type secureChannelMsg struct {
	SecurityHeader symmetricAlgorithmSecurityHeader
	SequenceHeader sequenceHeader
	TypeID         uatype.NodeId
	body           io.Reader
}

func (s secureChannelMsg) bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)

	if err := enc.Encode(s); err != nil {
		return nil, err
	}

	if _, err := io.Copy(&buf, s.body); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *secureChannelMsg) decodeHeader(r io.Reader) error {
	dec := binary.NewDecoder(r)
	return dec.Decode(s)
}

type asymmetricAlgorithmSecurityHeader struct {
	SecurityPolicyURI             string
	SenderCertificate             string
	ReceiverCertificateThumbprint string
}

// The SecurityPolicyURIs for supported asymmetric algorithms
const (
	SecurityPolicyURINone           string = "http://opcfoundation.org/UA/SecurityPolicy#None"
	SecurityPolicyURIBasic128Rsa15  string = "http://opcfoundation.org/UA/SecurityPolicy#Basic128Rsa15"
	SecurityPolicyURIBasic256       string = "http://opcfoundation.org/UA/SecurityPolicy#Basic256"
	SecurityPolicyURIBasic256Sha256 string = "http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256"
)

func asymmetricAlgorithmSecurityHeaderNone() asymmetricAlgorithmSecurityHeader {
	return asymmetricAlgorithmSecurityHeader{
		SecurityPolicyURI:             SecurityPolicyURINone,
		SenderCertificate:             "",
		ReceiverCertificateThumbprint: "",
	}
}

type symmetricAlgorithmSecurityHeader struct {
	TokenID uint32
}

type sequenceHeader struct {
	SequenceNumber uint32
	RequestID      uint32
}

/* --- Methods to provide default values --- */

func (s *SecureChannel) securityPolicyURI() string {
	if len(s.SecurityPolicyURI) > 0 {
		return s.SecurityPolicyURI
	}
	return DefaulSecurityPolicyURI
}

func (s *SecureChannel) lifeTime() uint32 {
	if s.LifeTime > 0 {
		return uint32(s.LifeTime.Seconds())
	}
	return uint32(DefaultLifeTime.Seconds())
}
