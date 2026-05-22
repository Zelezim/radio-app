package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// GET /programs -- listing with aggregated votes + ratings.
func (h *Handlers) ListPrograms(c echo.Context) error {
	var programs []models.Program
	if err := h.SB.Select("programs", "order=title.asc", &programs); err != nil {
		return c.String(http.StatusBadGateway, "failed to load programs: "+err.Error())
	}

	// Pull pre-aggregated totals from the SQL views and zip them onto the
	// programs. A small N here means we can afford two extra round-trips
	// instead of building a joined RPC.
	var totals []models.VoteTotal
	_ = h.SB.Select("vote_totals", "content_type=eq.program", &totals)
	totalsByID := map[string]models.VoteTotal{}
	for _, t := range totals {
		totalsByID[t.ContentID] = t
	}

	var ratings []models.ProgramRatingAvg
	_ = h.SB.Select("program_rating_avg", "", &ratings)
	ratingsByID := map[string]models.ProgramRatingAvg{}
	for _, r := range ratings {
		ratingsByID[r.ProgramID] = r
	}

	for i := range programs {
		if t, ok := totalsByID[programs[i].ID]; ok {
			programs[i].UpVotes = t.UpVotes
			programs[i].DownVotes = t.DownVotes
		}
		if r, ok := ratingsByID[programs[i].ID]; ok {
			programs[i].AvgRating = r.AvgScore
			programs[i].RatingCount = r.RatingCount
		}
	}

	data := struct {
		baseData
		Programs []models.Program
	}{baseData: h.base(c, "Programs"), Programs: programs}
	return c.Render(http.StatusOK, "programs.html", data)
}
