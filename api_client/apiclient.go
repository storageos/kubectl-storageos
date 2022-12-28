package apiclient

import (
	"context"
	"time"

	"github.com/storageos/kubectl-storageos/api_client/openapi"
)

// Client wraps the openAPI client providing a collection of useful methods such
// as GetEntityByName, making use of the underlying client endpoints and
// extending their logic with filters, objects transformations, etc.
//
// It also decorates the underlying openAPI Authenticate() method with cached
// auth capabilities, reusing an in-memory token (session) if it exists. If that
// fails, it will continue as normal and then cache the new token.
// This cache is used until Authenticate is called again (manually or when the
// reauth wrapper is triggered).
type Client struct {
	*openapi.OpenAPI

	cacheSessionToken     string
	cacheSessionExpiresAt time.Time
}

func NewClient(openapiClient *openapi.OpenAPI) Client {
	return Client{
		OpenAPI: openapiClient,
	}
}

func (c *Client) Authenticate(ctx context.Context, username, password string) (string, time.Time, error) {
	// TODO make configurable if we want to skip this feature of caching the session
	if c.cacheSessionToken != "" && c.cacheSessionExpiresAt.After(time.Now()) {
		err := c.UseAuthSession(ctx, c.cacheSessionToken)
		if err == nil {
			return c.cacheSessionToken, c.cacheSessionExpiresAt, nil
		}
		// failed to setup auth with in-memory session, proceed to create a new one
	}

	sessionToken, expiresBy, err := c.OpenAPI.Authenticate(ctx, username, password)
	if err != nil {
		return "", time.Now().Add(-time.Minute), err
	}

	// store new session in-memory
	c.cacheSessionToken = sessionToken
	c.cacheSessionExpiresAt = expiresBy

	return sessionToken, expiresBy, nil
}
