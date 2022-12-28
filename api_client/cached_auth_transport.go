package apiclient

import (
	"context"
	"time"
)

// AuthCachedTransport decorates the embedded transport with cached auth
// capabilities, modifying the Authenticate() behaviour. An instance will
// try to authenticate from the cache if the requested username is not the
// previous username requested for auth. If that fails, it will continue as
// normal, caching the result.
type AuthCachedTransport struct {
	Transport

	cacheSession AuthSession
}

// Authenticate attempts to fetch an authentication session for username from
// its cache, unless it is a repeated attempt. If the cache lookup fails or the
// call is a re-attempted auth then this authenticates as normal but caches
// the result.
func (co *AuthCachedTransport) Authenticate(ctx context.Context, username, password string) (AuthSession, error) {
	if co.cacheSession.Token != "" && co.cacheSession.ExpiresAt.After(time.Now()) {
		session, err := co.authenticateFromCache(ctx, username)
		if err == nil {
			return session, nil
		}
	}

	session, err := co.Transport.Authenticate(
		ctx,
		username,
		password,
	)
	if err != nil {
		return session, err
	}

	co.cacheSession = session

	return session, nil
}

// authenticateFromCache sets the inner transport to use the auth session
// previously created and stored in-memory.
func (co *AuthCachedTransport) authenticateFromCache(ctx context.Context, username string) (AuthSession, error) {
	return co.cacheSession, co.UseAuthSession(ctx, co.cacheSession)
}

// NewAuthCachedTransport decorates inner with use of an authentication session
// cache.
func NewAuthCachedTransport(inner Transport) *AuthCachedTransport {
	return &AuthCachedTransport{
		Transport: inner,
	}
}
