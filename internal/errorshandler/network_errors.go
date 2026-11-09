package errorshandler

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"syscall"
)

type NetworkErrorClassifier struct {
	BaseClassifier
}

func NewNetworkErrorClassifier() *NetworkErrorClassifier {
	return &NetworkErrorClassifier{
		BaseClassifier: NewBaseClassifier(classifyNetworkError),
	}
}

func classifyNetworkError(err error) Classification {
	if err == nil {
		return NotApplicable
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return Retriable
	}

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		if dnsError.IsTimeout || dnsError.IsTemporary {
			return Retriable
		}
		return NonRetriable
	}

	var (
		addressError       *net.AddrError
		invalidAddress     net.InvalidAddrError
		unknownAuthority   x509.UnknownAuthorityError
		invalidCertificate x509.CertificateInvalidError
		hostnameError      x509.HostnameError
	)

	if errors.As(err, &addressError) || errors.As(err, &invalidAddress) ||
		errors.As(err, &unknownAuthority) || errors.As(err, &invalidCertificate) || errors.As(err, &hostnameError) {
		return NonRetriable
	}

	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return Retriable
	}

	switch {
	case errors.Is(err, syscall.ECONNREFUSED),
		errors.Is(err, syscall.ECONNRESET),
		errors.Is(err, syscall.ECONNABORTED),
		errors.Is(err, syscall.ENETUNREACH),
		errors.Is(err, syscall.EHOSTUNREACH),
		errors.Is(err, syscall.EPIPE),
		errors.Is(err, net.ErrClosed),
		errors.Is(err, io.EOF),
		errors.Is(err, io.ErrUnexpectedEOF):
		return Retriable
	}

	return NotApplicable
}
