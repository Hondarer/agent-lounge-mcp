package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/hondarer/agent-lounge-mcp/internal/auth"
	"github.com/hondarer/agent-lounge-mcp/internal/config"
	"github.com/hondarer/agent-lounge-mcp/internal/store"
	"github.com/hondarer/agent-lounge-mcp/internal/tools"
)

// Bodies of the 401 responses. Each states the cause in one sentence, which an agent can pass on
// to its user. They never quote the token.
const (
	msgTokenMissing = "agent-lounge-mcp requires a token: register this server in the agent with the header " +
		"Authorization: Bearer <token>, using a token issued by agent-lounge-mcp user add."
	msgTokenInvalid = "agent-lounge-mcp rejected the token because it is not valid (it is mistyped, " +
		"or its user was removed or given a new token): ask the user to check the token registered for this server."
)

// checkHost limits the Host header of MCP requests. The check stops web pages that make a
// browser send requests to a loopback server under another name (DNS rebinding); cross-origin
// browser requests are rejected separately, in every configuration (see Handler).
//
//   - Listening only on loopback: the loopback names and addresses, and the names given with
//     --allowed-host (for a reverse proxy in front).
//   - Listening on an address other than loopback (--allow-remote): no check. The name that
//     reaches the server depends on the user's network, so no list could be right. Every
//     request needs a token there, and a page in a browser cannot attach one, so the requests
//     the check would stop are rejected by authenticate instead.
func checkHost(cfg config.Config, next http.Handler) http.Handler {
	if cfg.ListensRemote() {
		return next
	}
	allowed := map[string]bool{}
	for _, h := range cfg.AllowedHosts {
		allowed[strings.Trim(config.NormalizeHost(h), "[]")] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(config.NormalizeHost(host), "[]")
		if !config.IsLoopbackHost(host) && !allowed[host] {
			http.Error(w, fmt.Sprintf("Forbidden: invalid Host header %q", r.Host), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// userKey is the context key of the user authenticate decided for a request.
type userKey struct{}

// tokenHashKey is the context key of the hash of the token the request was authenticated
// with; absent for a request without a token.
type tokenHashKey struct{}

// authenticate decides the user of each MCP request from its Authorization header, looking the
// token up on every request, so that removing a user or rotating its token takes effect at the
// next request:
//
//   - a valid token: its user;
//   - no token, on a server that does not require one (see config.TokenRequired): the user local;
//   - no token, on a server that requires one: 401 with WWW-Authenticate: Bearer;
//   - a token that matches no user, or a header of another form: 401, whether or not a token
//     is required. A wrong token never falls back to local.
//
// The token itself does not travel further: the request goes on with a placeholder in its
// Authorization header (see bindUser).
func authenticate(cfg config.Config, sess Session, next http.Handler) http.Handler {
	required := cfg.TokenRequired()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, present, err := auth.BearerToken(r.Header)
		var user store.User
		var hash string
		switch {
		case err != nil:
			unauthorized(w, `Bearer error="invalid_token"`, msgTokenInvalid)
			return
		case !present && required:
			unauthorized(w, "Bearer", msgTokenMissing)
			return
		case !present:
			user = sess.LocalUser
		case !auth.WellFormed(token):
			unauthorized(w, `Bearer error="invalid_token"`, msgTokenInvalid)
			return
		default:
			hash = auth.Hash(token)
			user, err = sess.Store.UserByTokenHash(r.Context(), hash)
			switch {
			case errors.Is(err, store.ErrNoSuchUser):
				unauthorized(w, `Bearer error="invalid_token"`, msgTokenInvalid)
				return
			case err != nil:
				log.Printf("authenticate a request: %v", err)
				http.Error(w, "agent-lounge-mcp could not check the token: internal error.", http.StatusInternalServerError)
				return
			}
		}
		r = r.Clone(context.WithValue(context.WithValue(r.Context(), userKey{}, user), tokenHashKey{}, hash))
		r.Header.Set("Authorization", "Bearer "+placeholderToken)
		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter, challenge, msg string) {
	w.Header().Set("WWW-Authenticate", challenge)
	http.Error(w, msg, http.StatusUnauthorized)
}

// placeholderToken replaces the token in the requests passed to the SDK, which reads a bearer
// header (see bindUser) but must not see the secret.
const placeholderToken = "authenticated"

// bindUser hands the user decided by authenticate to the SDK as its token information
// (tools.TokenInfo). The SDK exposes no other way to attach it: its bearer token middleware
// is the one function that stores token information in a request, so bindUser uses that
// middleware with a verifier that returns the decided user and ignores the placeholder token.
//
// With the token information:
//   - the tools receive the user with every request (tools.Deps.RequestUser);
//   - the SDK binds each MCP connection (Mcp-Session-Id) to the user of its first request and
//     answers a request of another user on that connection with 403 Forbidden.
func bindUser(next http.Handler) http.Handler {
	verify := func(ctx context.Context, _ string, _ *http.Request) (*sdkauth.TokenInfo, error) {
		user, ok := ctx.Value(userKey{}).(store.User)
		if !ok {
			return nil, sdkauth.ErrInvalidToken
		}
		hash, _ := ctx.Value(tokenHashKey{}).(string)
		return tools.TokenInfo(user, hash), nil
	}
	// Tokens do not expire (a user's token is replaced with user rotate instead).
	return sdkauth.RequireBearerToken(verify, &sdkauth.RequireBearerTokenOptions{AllowMissingExpiration: true})(next)
}
