// Package oauth2cli provides better user experience on OAuth 2.0 and OpenID Connect (OIDC) on CLI.
// It allows simple and easy user interaction with Authorization Code Grant Flow and a local server.
package oauth2cli

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/xerrors"
)

var noopMiddleware = func(h http.Handler) http.Handler { return h }

// DefaultLocalServerSuccessHTML is a default response body on authorization success.
const DefaultLocalServerSuccessHTML = `<html><body>OK<script>window.close()</script></body></html>`

// Config represents a config for GetToken.
type Config struct {
	// OAuth2 config.
	// RedirectURL will be automatically set to the local server.
	OAuth2Config oauth2.Config

	// Options for an authorization request.
	// You can set oauth2.AccessTypeOffline and the PKCE options here.
	AuthCodeOptions []oauth2.AuthCodeOption
	// Options for a token request.
	// You can set the PKCE options here.
	TokenRequestOptions []oauth2.AuthCodeOption

	// Candidates of hostname and port which the local server binds to.
	// You can set port number to 0 to allocate a free port.
	// If multiple addresses are given, it will try the ports in order.
	// If nil or an empty slice is given, it defaults to "127.0.0.1:0" i.e. a free port.
	LocalServerBindAddress []string

	// A PEM-encoded certificate, and possibly the complete certificate chain.
	// When set, the server will serve TLS traffic using the specified
	// certificates. It's recommended that the public key's SANs contain
	// the loopback addresses - 'localhost', '127.0.0.1' and '::1'
	LocalServerCertFile string
	// A PEM-encoded private key for the certificate.
	// This is required when LocalServerCertFile is set.
	LocalServerKeyFile string

	// Response HTML body on authorization completed.
	// Default to DefaultLocalServerSuccessHTML.
	LocalServerSuccessHTML string
	// Middleware for the local server. Default to none.
	LocalServerMiddleware func(h http.Handler) http.Handler
	// A channel to send its URL when the local server is ready. Default to none.
	LocalServerReadyChan chan<- string

	// DEPRECATED: this will be removed in the future release.
	// Use LocalServerBindAddress instead.
	// Address which the local server binds to.
	// Default to "127.0.0.1".
	LocalServerAddress string
	// DEPRECATED: this will be removed in the future release.
	// Use LocalServerBindAddress instead.
	// Candidates of a port which the local server binds to.
	// If nil or an empty slice is given, LocalServerAddress is ignored and allocate a free port.
	// If multiple ports are given, they are appended to LocalServerBindAddress.
	LocalServerPort []int
}

func (c *Config) populateDeprecatedFields() {
	if len(c.LocalServerPort) > 0 {
		address := c.LocalServerAddress
		if address == "" {
			address = "127.0.0.1"
		}
		for _, port := range c.LocalServerPort {
			c.LocalServerBindAddress = append(c.LocalServerBindAddress, fmt.Sprintf("%s:%d", address, port))
		}
	}
}

// GetToken performs the Authorization Code Grant Flow and returns a token received from the provider.
// See https://tools.ietf.org/html/rfc6749#section-4.1
//
// This performs the following steps:
//
//	1. Start a local server at the port.
//	2. Open a browser and navigate it to the local server.
//	3. Wait for the user authorization.
// 	4. Receive a code via an authorization response (HTTP redirect).
// 	5. Exchange the code and a token.
// 	6. Return the code.
//
func GetToken(ctx context.Context, config Config) (*oauth2.Token, error) {
	if config.LocalServerMiddleware == nil {
		config.LocalServerMiddleware = noopMiddleware
	}
	if config.LocalServerSuccessHTML == "" {
		config.LocalServerSuccessHTML = DefaultLocalServerSuccessHTML
	}
	config.populateDeprecatedFields()
	code, err := receiveCodeViaLocalServer(ctx, &config)
	if err != nil {
		return nil, xerrors.Errorf("authorization error: %w", err)
	}
	token, err := config.OAuth2Config.Exchange(ctx, code, config.TokenRequestOptions...)
	if err != nil {
		return nil, xerrors.Errorf("could not exchange the code and token: %w", err)
	}
	return token, nil
}
