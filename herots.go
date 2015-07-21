/*
	HEROTS
	HERald Of The Swarm

	Package for fast TLS server creation.
*/
package herots

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

type Server struct {
	options *Options
	certs   struct {
		Cert tls.Certificate
		pool struct {
			IsSet bool
			Pool  *x509.CertPool
		}
	}
	listener       net.Listener
	logDestination io.Writer
}

// predefined errors messages
const (
	LoadKeyPairError      = "herots: load key pair error"
	LoadClientCaCertError = "herots srv: load client CA cert error"
	StartServerError      = "herots srv: start tls server error"
	NoKeyPairLoad         = "herots: no load key pair (use LoadKeyPair func)"
	AcceptConnError       = "herots srv: connection accept error"
)

/*
	A Options structure is used to configure a TLS server.
*/
type Options struct {
	// Server host.
	// By default server use "127.0.0.1".
	Host string

	// Server port.
	// By default server use "9000".
	Port int

	// LogLevel provides the opportunity to choose the level of
	// information messages.
	// Each level includes the messages from the previous level.
	// 0 - no messages
	// 1 - notice
	// 2 - info
	// 3 - error
	//
	// Default: '0'.
	LogLevel int

	// LogDestination provides the opportunity to choose the own
	// destination for log messages (errors, info, etc).
	//
	// Default: 'os.Stdout'.
	LogDestination io.Writer

	// See http://golang.org/pkg/crypto/tls/#ClientAuthType
	// By default server use tls.RequireAnyClientCert
	TLSAuthType tls.ClientAuthType
}

/*
	Return Server struct with predefined options.
*/
func NewServer() *Server {
	s := &Server{
		options: &Options{
			Host:        "127.0.0.1",
			Port:        9000,
			LogLevel:    0,
			TLSAuthType: tls.RequireAnyClientCert,
		},
	}
	s.logDestination = os.Stdout // send messages to stdout by default

	return s
}

// func for print messages
func (h *Server) log(m string, lvl int) {
	if h.options.LogLevel == 0 {
		return
	}

	if h.options.LogLevel <= lvl {
		fmt.Fprintf(h.logDestination, "herots srv: %s\n", m)
	}
}

/*
	Provides the opportunity to choose own destination for
	herots messages (errors, info, etc).

	By default server use os.Stdout.
*/
func (h *Server) SetMessagesDst(dst io.Writer) {
	h.logDestination = dst
}

/*
	Set herots server options (*Options).
*/
func (h *Server) Config(o *Options) {
	// check mandatory options
	if o.Port == 0 {
		h.log("port can't be '0'", 2)
		h.log("set port by default (9000)", 2)
		o.Port = 9000
	}

	h.options = o
}

/*
	Func for load certificate and private key pair.

	Public/private key pair require as PEM encoded data.
*/
func (h *Server) LoadKeyPair(cert, key []byte) error {
	// create cert pool
	h.certs.pool.Pool = x509.NewCertPool()

	// load keypair
	c, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return fmt.Errorf("%s: %v\n", LoadKeyPairError, err)
	}
	h.certs.Cert = c

	// add cert to pool
	pemData, _ := pem.Decode(cert)
	ca, err := x509.ParseCertificate(pemData.Bytes)
	if err != nil {
		return fmt.Errorf("%s: %v\n", LoadKeyPairError, err)
	}
	h.certs.pool.Pool.AddCert(ca)
	h.certs.pool.IsSet = true

	h.log("load key pair ok", 2)

	return nil
}

/*
	Add client CA certificate to x509.CertPool (tls.Config.ClientCAs).

	By default server add cert from server public/private key pair (LoadKeyPair)
	to cert pool.
*/
func (h *Server) AddClientCACert(cert []byte) error {
	pemData, _ := pem.Decode(cert)
	ca, err := x509.ParseCertificate(pemData.Bytes)
	if err != nil {
		return fmt.Errorf("%s: %v\n", LoadClientCaCertError, err)
	}
	h.certs.pool.Pool.AddCert(ca)

	h.log("load client CA cert ok", 2)

	return nil
}

/*
	Accept and return connection to server.
*/
func (h *Server) Accept() (net.Conn, error) {
	conn, err := h.listener.Accept()
	if err != nil {
		h.log("accept conn error: "+err.Error(), 3)
		return conn, fmt.Errorf("%s: %v\n", AcceptConnError, err)
	}
	h.log("accepted conn from "+conn.RemoteAddr().String(), 2)
	return conn, nil
}

/*
	Start server.
*/
func (h *Server) Start() error {
	// load keypair check
	if len(h.certs.Cert.Certificate) == 0 {
		return fmt.Errorf("%s\n", NoKeyPairLoad)
	}

	config := tls.Config{
		ClientAuth:   h.options.TLSAuthType,
		Certificates: []tls.Certificate{h.certs.Cert},
		ClientCAs:    h.certs.pool.Pool,
		Rand:         rand.Reader,
	}

	service := h.options.Host + ":" + strconv.Itoa(h.options.Port)

	listener, err := tls.Listen("tcp", service, &config)
	if err != nil {
		return fmt.Errorf("%s: %v\n", StartServerError, err)
	}
	h.listener = listener

	h.log("listening on "+service, 2)

	return nil
}
