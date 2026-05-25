package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/radioapp/radioapp/internal/models"
)

// GET /me -- a per-user activity page: ratings, votes, contact messages.
//
// Wired behind RequireAuth in main.go, so c.Get("session") is guaranteed
// non-nil here.
func (h *Handlers) MePage(c echo.Context) error {
	s := c.Get("session").(*models.Session)

	var (
		ratings  []models.MyRating
		votes    []models.MyVote
		messages []models.ContactRequest

		wg sync.WaitGroup
	)

	// Three independent reads, fanned out for latency.
	wg.Add(3)
	go func() {
		defer wg.Done()
		q := url.Values{}
		q.Set("user_id", "eq."+s.UserID)
		q.Set("select", "score,created_at,programs(id,title,image_url)")
		q.Set("order", "created_at.desc")
		_ = h.SB.Select("ratings", q.Encode(), &ratings)
	}()
	go func() {
		defer wg.Done()
		q := url.Values{}
		q.Set("user_id", "eq."+s.UserID)
		q.Set("content_type", "eq.program")
		q.Set("order", "created_at.desc")
		_ = h.SB.Select("votes", q.Encode(), &votes)
	}()
	go func() {
		defer wg.Done()
		q := url.Values{}
		q.Set("email", "eq."+s.Email)
		q.Set("order", "created_at.desc")
		q.Set("limit", "50") // keep page bounded
		_ = h.SB.Select("contact_requests", q.Encode(), &messages)
	}()
	wg.Wait()

	// Resolve program titles for the votes (no FK on content_id → programs,
	// so PostgREST won't embed it automatically).
	if len(votes) > 0 {
		ids := make([]string, 0, len(votes))
		seen := map[string]bool{}
		for _, v := range votes {
			if !seen[v.ContentID] {
				ids = append(ids, v.ContentID)
				seen[v.ContentID] = true
			}
		}
		var progs []models.Program
		q := url.Values{}
		q.Set("id", "in.("+strings.Join(ids, ",")+")")
		q.Set("select", "id,title,image_url")
		_ = h.SB.Select("programs", q.Encode(), &progs)
		byID := map[string]models.Program{}
		for _, p := range progs {
			byID[p.ID] = p
		}
		for i := range votes {
			if p, ok := byID[votes[i].ContentID]; ok {
				votes[i].ProgramTitle = p.Title
				votes[i].ProgramImage = p.ImageURL
			}
		}
	}

	data := struct {
		baseData
		Ratings  []models.MyRating
		Votes    []models.MyVote
		Messages []models.ContactRequest
	}{
		baseData: h.base(c, "My account"),
		Ratings:  ratings,
		Votes:    votes,
		Messages: messages,
	}
	return c.Render(http.StatusOK, "me.html", data)
}
