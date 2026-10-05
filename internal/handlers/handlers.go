// Package handlers contains the Echo handlers. They are grouped into a
// single Handlers struct so that the Supabase client and the session
// secret can be wired up once in main.go and shared across endpoints.
package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
	"github.com/radioapp/radioapp/internal/supabase"
)

// Handlers bundles the dependencies every handler needs.
type Handlers struct {
	SB                *supabase.Client
	SessionSecret     []byte
	StreamURL         string
	StreamURLFallback string // optional alternate format for <audio> source fallback
	StationName       string
}

const sessionCookieName = "rb_session"

// --- Signed cookie helpers ----------------------------------------------
//
// The cookie format is:   base64url(json(Session)) "." base64url(hmacSHA256)
//
// Tamper detection only — confidentiality is not required because the JWT
// inside is already a self-contained credential meant to be read by
// Supabase, not the user.

func (h *Handlers) sign(payload []byte) string {
	mac := hmac.New(sha256.New, h.SessionSecret)
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// encodeSession produces the cookie value for a Session.
func (h *Handlers) encodeSession(s models.Session) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(raw)
	return enc + "." + h.sign([]byte(enc)), nil
}

// decodeSession parses and validates a cookie value.
func (h *Handlers) decodeSession(value string) (*models.Session, error) {
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] == '.' {
			payload, sig := value[:i], value[i+1:]
			if !hmac.Equal([]byte(sig), []byte(h.sign([]byte(payload)))) {
				return nil, errors.New("invalid session signature")
			}
			raw, err := base64.RawURLEncoding.DecodeString(payload)
			if err != nil {
				return nil, err
			}
			var s models.Session
			if err := json.Unmarshal(raw, &s); err != nil {
				return nil, err
			}
			return &s, nil
		}
	}
	return nil, errors.New("malformed session cookie")
}

// setSessionCookie writes the signed cookie to the response.
func (h *Handlers) setSessionCookie(c echo.Context, s models.Session) error {
	v, err := h.encodeSession(s)
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    v,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
	return nil
}

// clearSessionCookie expires the cookie on the client.
func (h *Handlers) clearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// CurrentSession returns the logged-in session if the cookie is valid, else nil.
func (h *Handlers) CurrentSession(c echo.Context) *models.Session {
	cookie, err := c.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}
	s, err := h.decodeSession(cookie.Value)
	if err != nil {
		return nil
	}
	return s
}

// RequireAuth is Echo middleware that redirects unauthenticated users to /login
// (or returns 401 for HTMX/XHR requests).
func (h *Handlers) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		s := h.CurrentSession(c)
		if s == nil {
			if c.Request().Header.Get("HX-Request") == "true" {
				return c.String(http.StatusUnauthorized, "Please log in to continue.")
			}
			return c.Redirect(http.StatusFound, "/login")
		}
		c.Set("session", s)
		return next(c)
	}
}

// --- Template helpers ----------------------------------------------------

// baseData is the common payload every page template receives. Handlers may
// embed it into a larger struct via composition.
type baseData struct {
	Session           *models.Session
	StreamURL         string
	StreamURLFallback string
	StationName       string
	Title             string
	Flash             string
}

func (h *Handlers) base(c echo.Context, title string) baseData {
	return baseData{
		Session:           h.CurrentSession(c),
		StreamURL:         h.StreamURL,
		StreamURLFallback: h.StreamURLFallback,
		StationName:       h.StationName,
		Title:             title,
	}
}
