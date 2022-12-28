package apiclient

import (
	"context"
	"time"
)

// AuthSession encapsulates a re-usable auth session.
type AuthSession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// authCachedTransport decorates the embedded transport with cached auth
// capabilities, modifying the Authenticate() behaviour and instead reusing an
// in-memory session if exists.
// If that fails, it will continue as normal and then cache the new session.
type authCachedTransport struct {
	Transport

	cacheSession AuthSession
}

func (co *authCachedTransport) Authenticate(ctx context.Context, username, password string) (AuthSession, error) {
	if co.cacheSession.Token != "" && co.cacheSession.ExpiresAt.After(time.Now()) {
		err := co.UseAuthSession(ctx, co.cacheSession)
		if err == nil {
			return co.cacheSession, nil
		}
		// failed to setup auth with in-memory session, proceed to create a new one
	}

	session, err := co.Transport.Authenticate(ctx, username, password)
	if err != nil {
		return session, err
	}

	// store new session in-memory
	co.cacheSession = session

	return session, nil
}

// NewAuthCachedTransport decorates inner with use of an authentication session
// cache.
func NewAuthCachedTransport(inner Transport) *authCachedTransport {
	return &authCachedTransport{
		Transport: inner,
	}
}
