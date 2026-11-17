package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	cnAuthorizeURL = "https://chat.cyberneurova.ai/oauth/authorize"
	cnTokenURL     = "https://chat.cyberneurova.ai/api/oauth/token"
	cnUserinfoURL  = "https://chat.cyberneurova.ai/api/oauth/userinfo"
	oauthStateTTL  = 10 * time.Minute
)

// oauthStateStore holds in-flight PKCE authorization requests.
type oauthStateStore struct {
	mu sync.Mutex
	m  map[string]oauthPending
}

type oauthPending struct {
	verifier string
	expires  time.Time
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{m: make(map[string]oauthPending)}
}

func (ss *oauthStateStore) put(state, verifier string) {
	ss.mu.Lock()
	now := time.Now()
	for k, v := range ss.m {
		if now.After(v.expires) {
			delete(ss.m, k)
		}
	}
	ss.m[state] = oauthPending{verifier: verifier, expires: now.Add(oauthStateTTL)}
	ss.mu.Unlock()
}

func (ss *oauthStateStore) take(state string) (string, bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	p, ok := ss.m[state]
	if !ok {
		return "", false
	}
	delete(ss.m, state)
	if time.Now().After(p.expires) {
		return "", false
	}
	return p.verifier, true
}

func pkceVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func randOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// oauthEnabled reports whether all three OAuth env vars are set.
func (s *Server) oauthEnabled() bool {
	return s.oauthClientID != "" && s.oauthClientSecret != "" && s.oauthRedirectURI != ""
}

// GET /api/auth/cyberneurova — start the PKCE authorization flow.
func (s *Server) authCyberneurovaBegin(w http.ResponseWriter, r *http.Request) {
	if !s.oauthEnabled() {
		writeErr(w, 503, "Cyberneurova OAuth is not configured on this instance")
		return
	}
	verifier, err := pkceVerifier()
	if err != nil {
		writeErr(w, 500, "failed to generate PKCE verifier")
		return
	}
	state, err := randOAuthState()
	if err != nil {
		writeErr(w, 500, "failed to generate state")
		return
	}
	s.oauthStates.put(state, verifier)

	params := url.Values{
		"client_id":             {s.oauthClientID},
		"redirect_uri":          {s.oauthRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"profile email"},
		"state":                 {state},
		"code_challenge":        {pkceChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	http.Redirect(w, r, cnAuthorizeURL+"?"+params.Encode(), http.StatusFound)
}

// GET /api/auth/cyberneurova/callback — handle the redirect from Cyberneurova.
func (s *Server) authCyberneurovaCallback(w http.ResponseWriter, r *http.Request) {
	if !s.oauthEnabled() {
		writeErr(w, 503, "Cyberneurova OAuth is not configured on this instance")
		return
	}

	q := r.URL.Query()

	if errCode := q.Get("error"); errCode != "" {
		desc := q.Get("error_description")
		if desc == "" {
			desc = errCode
		}
		http.Redirect(w, r, "/login?error="+url.QueryEscape(desc), http.StatusFound)
		return
	}

	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("missing code or state"), http.StatusFound)
		return
	}

	verifier, ok := s.oauthStates.take(state)
	if !ok {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("invalid or expired state — please try again"), http.StatusFound)
		return
	}

	accessToken, err := s.exchangeOAuthCode(code, verifier)
	if err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("token exchange failed: "+err.Error()), http.StatusFound)
		return
	}

	if _, err := fetchCNUserinfo(accessToken); err != nil {
		http.Redirect(w, r, "/login?error="+url.QueryEscape("could not fetch user info"), http.StatusFound)
		return
	}

	tok, err := signJWT(s.jwtKey)
	if err != nil {
		writeErr(w, 500, "token generation failed")
		return
	}

	// Redirect to the frontend OAuth success page, which stores the token in localStorage.
	http.Redirect(w, r, "/oauth-callback?token="+url.QueryEscape(tok), http.StatusFound)
}

func (s *Server) exchangeOAuthCode(code, verifier string) (string, error) {
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.oauthRedirectURI},
		"client_id":     {s.oauthClientID},
		"client_secret": {s.oauthClientSecret},
		"code_verifier": {verifier},
	}
	resp, err := http.PostForm(cnTokenURL, body)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if tok.Error != "" {
		desc := tok.ErrorDesc
		if desc == "" {
			desc = tok.Error
		}
		return "", fmt.Errorf("%s", desc)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}
	return tok.AccessToken, nil
}

type cnUserinfo struct {
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func fetchCNUserinfo(accessToken string) (*cnUserinfo, error) {
	req, err := http.NewRequest(http.MethodGet, cnUserinfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint %d", resp.StatusCode)
	}
	var u cnUserinfo
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	if u.Sub == "" {
		return nil, fmt.Errorf("empty sub in userinfo response")
	}
	return &u, nil
}
