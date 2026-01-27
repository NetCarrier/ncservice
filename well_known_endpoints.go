package ncservice

import (
	"errors"
	"os"
	"strings"
)

// OauthEndpoints returns the standard JSON-ready data for oauth systems
// to coordinate authentication
func OauthEndpoints() (map[string]any, error) {
	issuer := os.Getenv("OAUTH_ISSUER")
	authz := os.Getenv("OAUTH_AUTHORIZATION_ENDPOINT")
	token := os.Getenv("OAUTH_TOKEN_ENDPOINT")
	jwks := os.Getenv("OAUTH_JWKS_URI")
	scopes := os.Getenv("OAUTH_SCOPES")

	if issuer == "" || authz == "" || token == "" || jwks == "" {
		return nil, errors.New("missing OIDC settings")
	}

	return map[string]interface{}{
		"issuer":                   issuer,
		"authorization_endpoint":   authz,
		"token_endpoint":           token,
		"jwks_uri":                 jwks,
		"response_types_supported": []string{"code"},
		"grant_types_supported":    []string{"authorization_code", "refresh_token"},
		"scopes_supported":         strings.Split(scopes, " "),
		"subject_types_supported":  []string{"public"},
	}, nil
}
