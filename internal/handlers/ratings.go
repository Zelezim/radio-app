package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// POST /rate -- save (or update) a 1..5 star rating.
//
// Expected form fields: program_id, score.
// Returns an updated "★ 4.2 (12)" fragment for HTMX swap.
func (h *Handlers) RateProgram(c echo.Context) error {
	s := c.Get("session").(*models.Session)

	programID := c.FormValue("program_id")
	scoreStr := c.FormValue("score")
	score, err := strconv.Atoi(scoreStr)
	if err != nil || score < 1 || score > 5 || programID == "" {
		return c.String(http.StatusBadRequest, "invalid rating payload")
	}

	rating := models.Rating{
		UserID:    s.UserID,
		ProgramID: programID,
		Score:     score,
	}
	if err := h.SB.Upsert("ratings", "user_id,program_id", []models.Rating{rating}, nil); err != nil {
		return c.String(http.StatusBadGateway, "could not save rating: "+err.Error())
	}

	// Pull the recomputed average from the SQL view.
	q := url.Values{}
	q.Set("program_id", "eq."+programID)
	var avgs []models.ProgramRatingAvg
	if err := h.SB.Select("program_rating_avg", q.Encode(), &avgs); err != nil {
		return c.String(http.StatusBadGateway, "could not read average: "+err.Error())
	}
	avg := 0.0
	count := 0
	if len(avgs) > 0 {
		avg = avgs[0].AvgScore
		count = avgs[0].RatingCount
	}

	frag := fmt.Sprintf(
		`<span class="rating-avg" data-avg="%.2f">★ %.1f <small>(%d)</small></span>`,
		avg, avg, count,
	)
	return c.HTML(http.StatusOK, frag)
}
