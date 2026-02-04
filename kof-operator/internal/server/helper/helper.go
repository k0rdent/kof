package helper

import (
	"context"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

type ContextKey string

const IdTokenContextKey ContextKey = "idToken"

func GetJwtTokenFromHeader(req *http.Request) string {
	authHeader := req.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

func GetIDToken(ctx context.Context) (*oidc.IDToken, bool) {
	idToken, ok := ctx.Value(IdTokenContextKey).(*oidc.IDToken)
	return idToken, ok
}
