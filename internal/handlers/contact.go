package handlers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// POST /contact -- submission from the floating dock form.
//
// The form lives in base.html and is submitted via HTMX. We respond with
// a small HTML fragment that gets swapped into #contactResult, so the
// user never leaves the current page.
//
// request_type is a multi-value checkbox group. We persist all selected
// types as a comma-separated string in the existing TEXT column.
func (h *Handlers) SubmitContact(c echo.Context) error {
	// ParseForm populates PostForm so we can read the repeated field.
	_ = c.Request().ParseForm()

	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	if name == "" || email == "" {
		return c.HTML(http.StatusBadRequest,
			`<span class="msg error">Name and email are required.</span>`)
	}

	types := c.Request().PostForm["request_type"]
	requestType := strings.Join(types, ",")

	req := models.ContactRequest{
		Name:        name,
		Email:       email,
		Phone:       strings.TrimSpace(c.FormValue("phone")),
		Address:     strings.TrimSpace(c.FormValue("address")),
		Message:     strings.TrimSpace(c.FormValue("message")),
		RequestType: requestType,
		Status:      "new",
	}
	if err := h.SB.Insert("contact_requests", []models.ContactRequest{req}, nil); err != nil {
		return c.HTML(http.StatusBadGateway,
			`<span class="msg error">Could not save. Please try again.</span>`)
	}

	return c.HTML(http.StatusOK,
		`<span class="msg success">✅ Thanks! We received your message.</span>`)
}
