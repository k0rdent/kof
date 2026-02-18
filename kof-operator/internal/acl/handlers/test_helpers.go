package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// MockIDToken creates a properly signed JWT token that can be parsed by oidc.IDToken
func MockIDToken(claims map[string]any) *oidc.IDToken {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	cl := jwt.Claims{
		Subject:   "test-user",
		Issuer:    "https://test-issuer.example.com",
		Audience:  jwt.Audience{"test-audience"},
		Expiry:    jwt.NewNumericDate(now.Add(time.Hour)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	builder := jwt.Signed(sig).Claims(cl).Claims(claims)
	raw, err := builder.Serialize()
	if err != nil {
		panic(err)
	}

	keySet := &staticKeySet{key: &privateKey.PublicKey}
	verifier := oidc.NewVerifier(
		"https://test-issuer.example.com",
		keySet,
		&oidc.Config{SkipClientIDCheck: true, SkipExpiryCheck: true},
	)

	idToken, err := verifier.Verify(context.Background(), raw)
	if err != nil {
		panic(err)
	}

	return idToken
}

// staticKeySet implements oidc.KeySet for testing
type staticKeySet struct {
	key *rsa.PublicKey
}

func (s *staticKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	parsed, err := jose.ParseSigned(jwt, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return nil, err
	}
	return parsed.Verify(s.key)
}
