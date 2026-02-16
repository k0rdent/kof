package helper

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type ContextKey string

const IdTokenContextKey ContextKey = "idToken"

func GetJwtTokenFromHeader(req *http.Request) string {
	authHeader := req.Header.Get("Authorization")
	if token, found := strings.CutPrefix(authHeader, "Bearer "); found {
		return token
	}
	return ""
}

func GetIDToken(ctx context.Context) (*oidc.IDToken, bool) {
	idToken, ok := ctx.Value(IdTokenContextKey).(*oidc.IDToken)
	return idToken, ok
}
