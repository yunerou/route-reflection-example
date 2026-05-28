package geminivertexchatadapter

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const vertexScope = "https://www.googleapis.com/auth/cloud-platform"

// newTokenSource returns an oauth2.TokenSource using either an explicit
// service-account JSON file or Application Default Credentials.
func newTokenSource(ctx context.Context, credentialsFile *string) (oauth2.TokenSource, error) {
	if credentialsFile != nil && *credentialsFile != "" {
		data, err := os.ReadFile(*credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("gemini-vertex-chat: read credentials file %q: %w", *credentialsFile, err)
		}
		creds, err := google.CredentialsFromJSON(ctx, data, vertexScope)
		if err != nil {
			return nil, fmt.Errorf("gemini-vertex-chat: parse credentials JSON: %w", err)
		}
		return oauth2.ReuseTokenSource(nil, creds.TokenSource), nil
	}
	creds, err := google.FindDefaultCredentials(ctx, vertexScope)
	if err != nil {
		return nil, fmt.Errorf("gemini-vertex-chat: find default credentials: %w", err)
	}
	return oauth2.ReuseTokenSource(nil, creds.TokenSource), nil
}
