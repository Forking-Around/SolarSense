package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const tokenCookie = "solarsense_access"

type Supabase struct {
	URL, PublishableKey, PublicURL string
	HTTP                           *http.Client
}
type User struct{ ID, Email string }
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s Supabase) Enabled() bool { return s.URL != "" && s.PublishableKey != "" }
func (s Supabase) Begin(w http.ResponseWriter, r *http.Request) error {
	verifier, err := random(48)
	if err != nil {
		return err
	}
	state, err := random(24)
	if err != nil {
		return err
	}
	secure := strings.HasPrefix(s.PublicURL, "https://")
	http.SetCookie(w, &http.Cookie{Name: "ss_pkce", Value: verifier, Path: "/auth/callback", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.SetCookie(w, &http.Cookie{Name: "ss_state", Value: state, Path: "/auth/callback", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	callback := strings.TrimRight(s.PublicURL, "/") + "/auth/callback?state=" + url.QueryEscape(state)
	q := url.Values{"provider": {"google"}, "redirect_to": {callback}, "code_challenge": {challenge}, "code_challenge_method": {"s256"}}
	http.Redirect(w, r, strings.TrimRight(s.URL, "/")+"/auth/v1/authorize?"+q.Encode(), http.StatusSeeOther)
	return nil
}
func (s Supabase) Callback(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	vc, e1 := r.Cookie("ss_pkce")
	sc, e2 := r.Cookie("ss_state")
	if e1 != nil || e2 != nil || r.URL.Query().Get("state") != sc.Value {
		return fmt.Errorf("invalid or expired OAuth state")
	}
	body, _ := json.Marshal(map[string]string{"auth_code": r.URL.Query().Get("code"), "code_verifier": vc.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.URL, "/")+"/auth/v1/token?grant_type=pkce", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", s.PublishableKey)
	resp, err := s.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Supabase token exchange returned %s", resp.Status)
	}
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return err
	}
	if tok.AccessToken == "" {
		return fmt.Errorf("Supabase returned no access token")
	}
	secure := strings.HasPrefix(s.PublicURL, "https://")
	http.SetCookie(w, &http.Cookie{Name: tokenCookie, Value: tok.AccessToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: tok.ExpiresIn})
	return nil
}
func (s Supabase) User(ctx context.Context, r *http.Request) (User, error) {
	token, err := AccessToken(r)
	if err != nil {
		return User{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.URL, "/")+"/auth/v1/user", nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("apikey", s.PublishableKey)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client().Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("unauthenticated")
	}
	var x struct{ ID, Email string }
	if err := json.NewDecoder(resp.Body).Decode(&x); err != nil {
		return User{}, err
	}
	return User{ID: x.ID, Email: x.Email}, nil
}

func AccessToken(r *http.Request) (string, error) {
	c, err := r.Cookie(tokenCookie)
	if err != nil {
		return "", err
	}
	if c.Value == "" {
		return "", fmt.Errorf("empty access token")
	}
	return c.Value, nil
}
func (s Supabase) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 15 * time.Second}
}
func random(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
