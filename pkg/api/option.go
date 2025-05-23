package api

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/optiflow-os/tracelens-cli/pkg/heartbeat"
	"github.com/optiflow-os/tracelens-cli/pkg/log"

	"github.com/Azure/go-ntlmssp"
)

// Option is a functional option for Client.
type Option func(*Client)

// WithAuth adds authentication via Authorization header.
func WithAuth(auth BasicAuth) (Option, error) {
	authHeaderValue, err := auth.HeaderValue()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth header value: %w", err)
	}

	return func(c *Client) {
		next := c.doFunc
		c.doFunc = func(c *Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("Authorization", authHeaderValue)
			return next(c, req)
		}
	}, nil
}

// WithDisableSSLVerify disables verification of insecure certificates.
func WithDisableSSLVerify() Option {
	return func(c *Client) {
		transport := LazyCreateNewTransport(c)

		tlsConfig := transport.TLSClientConfig
		tlsConfig.InsecureSkipVerify = true

		transport.TLSClientConfig = tlsConfig
		c.client.Transport = transport
	}
}

// WithNTLM allows authentication via ntlm protocol.
func WithNTLM(creds string) (Option, error) {
	if !strings.Contains(creds, `\\`) {
		return Option(func(*Client) {}), fmt.Errorf("invalid ntlm credentials format %q. does not contain '\\\\'", creds)
	}

	splitted := strings.Split(creds, ":")

	auth := BasicAuth{
		User: splitted[0],
	}

	if len(splitted) == 2 {
		auth.Secret = splitted[1]
	}

	withAuth, err := WithAuth(auth)
	if err != nil {
		return Option(func(*Client) {}), err
	}

	return func(c *Client) {
		withAuth(c)

		c.client.Transport = ntlmssp.Negotiator{
			RoundTripper: LazyCreateNewTransport(c),
		}
	}, nil
}

// WithNTLMRequestRetry will, upon request failure, retry with ntlm authentication.
func WithNTLMRequestRetry(ctx context.Context, creds string) (Option, error) {
	withNTLM, err := WithNTLM(creds)
	if err != nil {
		return Option(func(*Client) {}), err
	}

	return func(c *Client) {
		logger := log.Extract(ctx)

		next := c.doFunc
		c.doFunc = func(cl *Client, req *http.Request) (*http.Response, error) {
			resp, err := next(c, req)
			if err != nil {
				logger.Errorf("request to api failed with error %q. Will retry with ntlm auth", err)

				clCopy := cl
				withNTLM(clCopy)

				return clCopy.Do(ctx, req)
			}

			return resp, nil
		}
	}, nil
}

// WithProxy configures the client to proxy outgoing requests to the specified url.
func WithProxy(proxyURL string) (Option, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy url %q: %s", proxyURL, err)
	}

	return func(c *Client) {
		transport := LazyCreateNewTransport(c)
		transport.Proxy = http.ProxyURL(u)
		c.client.Transport = transport
	}, nil
}

// WithSSLCertFile overrides the default CA certs file to trust specified cert file.
func WithSSLCertFile(filepath string) (Option, error) {
	caCert, err := os.ReadFile(filepath) // nolint:gosec
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return WithSSLCertPool(caCertPool), nil
}

// WithSSLCertPool overrides the default CA cert pool to trust specified cert pool.
func WithSSLCertPool(caCertPool *x509.CertPool) Option {
	return func(c *Client) {
		transport := LazyCreateNewTransport(c)

		tlsConfig := transport.TLSClientConfig
		tlsConfig.RootCAs = caCertPool

		transport.TLSClientConfig = tlsConfig

		c.client.Transport = transport
	}
}

// WithTimeout configures a timeout for all requests.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithHostname sets the X-Machine-Name header to the passed in hostname.
func WithHostname(hostname string) Option {
	return func(c *Client) {
		next := c.doFunc
		c.doFunc = func(c *Client, req *http.Request) (*http.Response, error) {
			hostname = url.QueryEscape(hostname)
			req.Header.Set("X-Machine-Name", hostname)

			return next(c, req)
		}
	}
}

// WithTimezone sets the TimeZone header to the passed in timezone.
func WithTimezone(timezone string) Option {
	return func(c *Client) {
		next := c.doFunc
		c.doFunc = func(c *Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("Timezone", timezone)

			return next(c, req)
		}
	}
}

// WithUserAgent sets the User-Agent header on all requests, including the passed
// in value for plugin.
func WithUserAgent(ctx context.Context, plugin string) Option {
	userAgent := heartbeat.UserAgent(ctx, plugin)

	return func(c *Client) {
		next := c.doFunc
		c.doFunc = func(c *Client, req *http.Request) (*http.Response, error) {
			req.Header.Set("User-Agent", userAgent)

			return next(c, req)
		}
	}
}
