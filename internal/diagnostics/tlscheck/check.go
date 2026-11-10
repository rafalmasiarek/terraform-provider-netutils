package tlscheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// Result is the outcome of a single TLS certificate inspection.
type Result struct {
	Success         bool
	Error           string
	NotBefore       time.Time
	NotAfter        time.Time
	Issuer          string
	Subject         string
	DNSNames        []string
	Expired         bool
	DaysUntilExpiry int64
}

// Check connects to host:port and inspects the leaf certificate presented
// in the TLS handshake. InsecureSkipVerify is used deliberately - this is
// a diagnostic inspecting whatever certificate is actually there
// (including an expired or otherwise invalid one), not a security
// decision about whether to trust it.
func Check(ctx context.Context, host string, port int, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return Result{Success: false, Error: err.Error()}
	}
	defer rawConn.Close()

	tlsConn := tls.Client(rawConn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	defer tlsConn.Close()

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return Result{Success: false, Error: err.Error()}
	}

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return Result{Success: false, Error: "server presented no certificate"}
	}
	leaf := certs[0]

	now := time.Now()
	expired := now.After(leaf.NotAfter)
	daysUntilExpiry := int64(leaf.NotAfter.Sub(now).Hours() / 24)

	return Result{
		Success:         true,
		NotBefore:       leaf.NotBefore,
		NotAfter:        leaf.NotAfter,
		Issuer:          leaf.Issuer.String(),
		Subject:         leaf.Subject.String(),
		DNSNames:        leaf.DNSNames,
		Expired:         expired,
		DaysUntilExpiry: daysUntilExpiry,
	}
}
