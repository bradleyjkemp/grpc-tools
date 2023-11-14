package grpc_proxy

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/bradleyjkemp/grpc-tools/testutils"

	"golang.org/x/net/http2"

	"github.com/bradleyjkemp/grpc-tools/internal/tlsmux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestTLSMux_HTTP2Support verifies that HTTP/2 is supported.
func TestTLSMux_HTTP2Support(t *testing.T) {
	// New logger.
	logger := logrus.New()
	// New proxy listener.
	ln, err := net.Listen("tcp", "localhost:0")
	listenerPort := ln.Addr().(*net.TCPAddr).Port
	require.NoErrorf(t, err, "failed creating tcp listener on port %d", listenerPort)
	proxyLis := newProxyListener(logger, ln)

	// New keypair.
	tlsCert, err := testutils.NewSelfSignedKeyPair()
	require.NoError(t, err, "failed loading X509 keypair")
	x509Cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	require.NoError(t, err, "failed parsing certificate out of keypair")

	// Get TLS listener.
	_, httpsLis := tlsmux.New(logger, proxyLis, x509Cert, tlsCert, ioutil.Discard)

	// Start mock server with TLS listener.
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Listener = httpsLis
	server.Start()

	// Request the server with HTTP2 and make sure the connection passes.
	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest("POST", "https://localhost:"+strconv.Itoa(listenerPort), strings.NewReader(""))
	require.NoError(t, err, "failed creating request object")
	_, err = client.Do(req)
	require.NoError(t, err, "failed requesting")
}
