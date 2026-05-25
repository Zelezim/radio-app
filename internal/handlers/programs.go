package handlers

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// GET /programs -- listing with aggregated votes + ratings.
//
// All three Supabase REST calls (programs, vote_totals, program_rating_avg)
// are independent, so we fire them in parallel goroutines and collect the
// results before stitching everything together. Cuts the wall-clock time
// from sum(latencies) to max(latencies) — typically ~3x faster on Render's
// free tier where each round-trip is ~300 ms.
func (h *Handlers) ListPrograms(c echo.Context) error {
	var (
		programs []models.Program
		totals   []models.VoteTotal
		ratings  []models.ProgramRatingAvg

		programsErr error // only the programs query is fatal — the others
		// degrade gracefully (a missing aggregate just shows 0).

		wg sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		programsErr = h.SB.Select("programs", "order=title.asc", &programs)
	}()
	go func() {
		defer wg.Done()
		_ = h.SB.Select("vote_totals", "content_type=eq.program", &totals)
	}()
	go func() {
		defer wg.Done()
		_ = h.SB.Select("program_rating_avg", "", &ratings)
	}()
	wg.Wait()

	if programsErr != nil {
		return c.String(http.StatusBadGateway, "failed to load programs: "+programsErr.Error())
	}

	// Zip the aggregates onto each program by id.
	totalsByID := map[string]models.VoteTotal{}
	for _, t := range totals {
		totalsByID[t.ContentID] = t
	}
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
