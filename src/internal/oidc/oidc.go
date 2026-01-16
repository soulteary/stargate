package oidc

import (
	"context"
	"errors"
	"fmt"

	oidcext "github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Provider represents an OIDC provider configuration
type Provider struct {
	issuerURL    string
	clientID     string
	clientSecret string
	redirectURI  string
	verifier     *oidcext.IDTokenVerifier
	oauth2Config *oauth2.Config
	provider     *oidcext.Provider
}

// NewProvider creates a new OIDC provider instance
func NewProvider(issuerURL, clientID, clientSecret, redirectURI string) (*Provider, error) {
	if issuerURL == "" {
		return nil, errors.New("issuer URL is required")
	}
	if clientID == "" {
		return nil, errors.New("client ID is required")
	}
	if clientSecret == "" {
		return nil, errors.New("client secret is required")
	}

	logrus.Infof("Initializing OIDC provider with issuer: %s", issuerURL)

	ctx := context.Background()
	provider, err := oidcext.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidcext.Config{
		ClientID: clientID,
	})

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidcext.ScopeOpenID, "email"},
	}
	if redirectURI != "" {
		oauth2Config.RedirectURL = redirectURI
	}

	return &Provider{
		issuerURL:    issuerURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		provider:     provider,
	}, nil
}

// AuthURL generates the OAuth2 authorization URL
func (p *Provider) AuthURL(state string) string {
	return p.oauth2Config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for an OAuth2 token
func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies the ID token and returns the claims
func (p *Provider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidcext.IDToken, error) {
	return p.verifier.Verify(ctx, rawIDToken)
}

// GetIssuer returns the issuer URL
func (p *Provider) GetIssuer() string {
	return p.issuerURL
}
