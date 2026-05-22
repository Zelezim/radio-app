package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GET / -- the home page with the live player.
func (h *Handlers) Home(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", h.base(c, "Live"))
}
