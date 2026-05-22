package handlers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// POST /vote -- cast / replace a vote.
//
// Expected form fields (HTMX-friendly):
//   content_id   uuid
//   content_type "program" | "music"
//   value        "1" | "-1"
//
// Replies with the updated tally as a small HTML fragment so HTMX can
// hx-swap it straight into the card.
func (h *Handlers) CastVote(c echo.Context) error {
	s := c.Get("session").(*models.Session)

	contentID := c.FormValue("content_id")
	contentType := c.FormValue("content_type")
	value := c.FormValue("value")
	if contentID == "" || (contentType != "program" && contentType != "music") || (value != "1" && value != "-1") {
		return c.String(http.StatusBadRequest, "invalid vote payload")
	}
	intValue := 1
	if value == "-1" {
		intValue = -1
	}

	vote := models.Vote{
		UserID:      s.UserID,
		ContentID:   contentID,
		ContentType: contentType,
		Value:       intValue,
	}
	if err := h.SB.Upsert("votes", "user_id,content_id,content_type", []models.Vote{vote}, nil); err != nil {
		return c.String(http.StatusBadGateway, "could not save vote: "+err.Error())
	}

	// Re-read the totals so the client always sees a fresh count.
	q := url.Values{}
	q.Set("content_id", "eq."+contentID)
	q.Set("content_type", "eq."+contentType)
	var totals []models.VoteTotal
	if err := h.SB.Select("vote_totals", q.Encode(), &totals); err != nil {
		return c.String(http.StatusBadGateway, "could not read totals: "+err.Error())
	}
	up, down := 0, 0
	if len(totals) > 0 {
		up, down = totals[0].UpVotes, totals[0].DownVotes
	}

	frag := fmt.Sprintf(
		`<span class="vote-tally" data-up="%d" data-down="%d">👍 %d · 👎 %d</span>`,
		up, down, up, down,
	)
	return c.HTML(http.StatusOK, frag)
}
