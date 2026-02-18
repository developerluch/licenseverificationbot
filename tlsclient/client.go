package tlsclient

import (
	tls_client "github.com/bogdanfinn/tls-client"
	tls_client_profiles "github.com/bogdanfinn/tls-client/profiles"
)

// Client is a factory for creating TLS client sessions with Chrome fingerprints.
type Client struct{}

// New creates a new TLS client factory.
func New() *Client {
	return &Client{}
}

// NewSession creates a fresh HTTP client with an isolated cookie jar and Chrome_124 fingerprint.
// Each scraper call should get a fresh session for cookie isolation.
func (c *Client) NewSession() (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(90),
		tls_client.WithClientProfile(tls_client_profiles.Chrome_124),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(nil, options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}
