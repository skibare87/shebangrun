package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type OAuthProvider struct {
	config *oauth2.Config
	userInfoURL string
}

func NewGitHubProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
		userInfoURL: "https://api.github.com/user",
	}
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *OAuthProvider {
	return &OAuthProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		userInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
	}
}

func (p *OAuthProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state)
}

func (p *OAuthProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

type OAuthUser struct {
	ID       string `json:"-"` // Custom unmarshal
	Email    string `json:"email"`
	Username string `json:"login"` // GitHub uses "login"
	Name     string `json:"name"`
	RawID    interface{} `json:"id"` // Can be string or number
}

func (u *OAuthUser) UnmarshalJSON(data []byte) error {
	type Alias OAuthUser
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	// Convert RawID to string
	switch v := u.RawID.(type) {
	case string:
		u.ID = v
	case float64:
		u.ID = fmt.Sprintf("%.0f", v)
	case int:
		u.ID = fmt.Sprintf("%d", v)
	}
	
	return nil
}

func (p *OAuthProvider) GetUserInfo(ctx context.Context, token *oauth2.Token) (*OAuthUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", body)
	}
	
	var user OAuthUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	
	// If email is empty (GitHub private email), fetch from emails API
	if user.Email == "" && p.userInfoURL == "https://api.github.com/user" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if json.NewDecoder(emailResp.Body).Decode(&emails) == nil {
				for _, e := range emails {
					if e.Primary {
						user.Email = e.Email
						break
					}
				}
				if user.Email == "" && len(emails) > 0 {
					user.Email = emails[0].Email
				}
			}
		}
	}
	
	return &user, nil
}
