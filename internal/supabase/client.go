// Package supabase is a tiny HTTP wrapper around the Supabase Auth (GoTrue)
// and PostgREST endpoints. We intentionally avoid the official SDK to keep
// the dependency surface tiny.
package supabase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a single Supabase project.
//
// Two keys are kept:
//   - AnonKey is used during signup / signin (GoTrue requires the anon key
//     as the apikey header).
//   - SecretKey (service role) is used for trusted backend reads/writes
//     that bypass row level security.
type Client struct {
	BaseURL    string
	AnonKey    string
	SecretKey  string
	HTTPClient *http.Client
}

// New builds a client with sensible HTTP timeouts.
func New(baseURL, anonKey, secretKey string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AnonKey:    anonKey,
		SecretKey:  secretKey,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Auth (GoTrue) -------------------------------------------------------

// AuthUser is the subset of the GoTrue user payload we care about.
type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// AuthSession is what /auth/v1/token returns on a successful sign-in.
type AuthSession struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	User         AuthUser `json:"user"`
}

// authError matches the GoTrue error shape.
type authError struct {
	Code             int    `json:"code"`
	Msg              string `json:"msg"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// SignUp creates a new user and (depending on project settings) returns a
// session. With email confirmation enabled, AccessToken will be empty.
func (c *Client) SignUp(email, password string) (*AuthSession, error) {
	body := map[string]string{"email": email, "password": password}
	var out AuthSession
	if err := c.doAuth("POST", "/auth/v1/signup", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SignIn exchanges email+password for an access_token.
func (c *Client) SignIn(email, password string) (*AuthSession, error) {
	body := map[string]string{"email": email, "password": password}
	var out AuthSession
	if err := c.doAuth("POST", "/auth/v1/token?grant_type=password", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SignOut invalidates the access_token on the Supabase side. It is safe to
// ignore the error: the user will be logged out locally either way.
func (c *Client) SignOut(accessToken string) error {
	req, err := http.NewRequest("POST", c.BaseURL+"/auth/v1/logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.AnonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// doAuth is a small helper that hits GoTrue and JSON-decodes the response.
func (c *Client) doAuth(method, path string, body any, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.AnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var ae authError
		_ = json.Unmarshal(raw, &ae)
		msg := ae.Msg
		if msg == "" {
			msg = ae.ErrorDescription
		}
		if msg == "" {
			msg = ae.Error
		}
		if msg == "" {
			msg = string(raw)
		}
		return fmt.Errorf("supabase auth %d: %s", resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// --- PostgREST -----------------------------------------------------------

// Select runs a GET against /rest/v1/<table> using the service role key.
// `query` is the URL-encoded PostgREST query string (without the leading ?).
func (c *Client) Select(table, query string, out any) error {
	u := fmt.Sprintf("%s/rest/v1/%s", c.BaseURL, table)
	if query != "" {
		u += "?" + query
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)
	return c.doJSON(req, out)
}

// Insert POSTs `payload` (a struct or slice of structs) into `table`.
// If `returning` is non-nil it must be a pointer to a slice; the inserted
// rows are decoded into it.
func (c *Client) Insert(table string, payload any, returning any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.BaseURL+"/rest/v1/"+table, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	if returning != nil {
		req.Header.Set("Prefer", "return=representation")
	} else {
		req.Header.Set("Prefer", "return=minimal")
	}
	return c.doJSON(req, returning)
}

// Upsert POSTs with on_conflict resolution. `onConflict` is a comma-separated
// list of columns that define uniqueness (e.g. "user_id,program_id").
func (c *Client) Upsert(table, onConflict string, payload any, returning any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("on_conflict", onConflict)
	req, err := http.NewRequest("POST", c.BaseURL+"/rest/v1/"+table+"?"+q.Encode(), bytes.NewReader(buf))
	if err != nil {
		return err
	}
	c.setServiceHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	prefer := "resolution=merge-duplicates,return=minimal"
	if returning != nil {
		prefer = "resolution=merge-duplicates,return=representation"
	}
	req.Header.Set("Prefer", prefer)
	return c.doJSON(req, returning)
}

// setServiceHeaders attaches the service role key for trusted server-side
// calls. RLS is bypassed when using this key.
func (c *Client) setServiceHeaders(req *http.Request) {
	req.Header.Set("apikey", c.SecretKey)
	req.Header.Set("Authorization", "Bearer "+c.SecretKey)
	req.Header.Set("Accept", "application/json")
}

// doJSON executes the request and decodes a JSON body into out (if non-nil).
func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("supabase %s %s -> %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return errors.New("decode response: " + err.Error())
	}
	return nil
}
