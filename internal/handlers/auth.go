package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// GET /login
func (h *Handlers) LoginPage(c echo.Context) error {
	if h.CurrentSession(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	data := struct {
		baseData
		Error string
	}{baseData: h.base(c, "Login")}
	return c.Render(http.StatusOK, "login.html", data)
}

// POST /login
func (h *Handlers) Login(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	session, err := h.SB.SignIn(email, password)
	if err != nil {
		data := struct {
			baseData
			Error string
		}{baseData: h.base(c, "Login"), Error: friendlyAuthError(err)}
		return c.Render(http.StatusUnauthorized, "login.html", data)
	}

	if err := h.setSessionCookie(c, models.Session{
		UserID:      session.User.ID,
		Email:       session.User.Email,
		AccessToken: session.AccessToken,
	}); err != nil {
		return c.String(http.StatusInternalServerError, "session error")
	}
	return c.Redirect(http.StatusFound, "/")
}

// GET /register
func (h *Handlers) RegisterPage(c echo.Context) error {
	if h.CurrentSession(c) != nil {
		return c.Redirect(http.StatusFound, "/")
	}
	data := struct {
		baseData
		Error   string
		Success string
	}{baseData: h.base(c, "Create account")}
	return c.Render(http.StatusOK, "register.html", data)
}

// POST /register
func (h *Handlers) Register(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	fullName := strings.TrimSpace(c.FormValue("full_name"))

	if len(password) < 6 {
		data := struct {
			baseData
			Error   string
			Success string
		}{baseData: h.base(c, "Create account"), Error: "Password must be at least 6 characters."}
		return c.Render(http.StatusBadRequest, "register.html", data)
	}

	session, err := h.SB.SignUp(email, password)
	if err != nil {
		data := struct {
			baseData
			Error   string
			Success string
		}{baseData: h.base(c, "Create account"), Error: friendlyAuthError(err)}
		return c.Render(http.StatusBadRequest, "register.html", data)
	}

	// Create the matching profile row so foreign keys (votes, ratings)
	// resolve. We swallow the error because profiles is optional metadata.
	if session.User.ID != "" {
		_ = h.SB.Insert("profiles", []map[string]any{{
			"id":        session.User.ID,
			"full_name": fullName,
		}}, nil)
	}

	// If the project requires email confirmation the access_token is empty.
	if session.AccessToken == "" {
		data := struct {
			baseData
			Error   string
			Success string
		}{
			baseData: h.base(c, "Create account"),
			Success:  "Account created. Check your inbox to confirm your email, then log in.",
		}
		return c.Render(http.StatusOK, "register.html", data)
	}

	if err := h.setSessionCookie(c, models.Session{
		UserID:      session.User.ID,
		Email:       session.User.Email,
		AccessToken: session.AccessToken,
	}); err != nil {
		return c.String(http.StatusInternalServerError, "session error")
	}
	return c.Redirect(http.StatusFound, "/")
}

// POST /logout
func (h *Handlers) Logout(c echo.Context) error {
	if s := h.CurrentSession(c); s != nil {
		_ = h.SB.SignOut(s.AccessToken) // best-effort
	}
	h.clearSessionCookie(c)
	return c.Redirect(http.StatusFound, "/login")
}

// friendlyAuthError trims the noisy Supabase wrapper around an auth failure.
func friendlyAuthError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	}
	if msg == "" {
		msg = "Authentication failed."
	}
	return msg
}
