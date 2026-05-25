// Package models defines the shared domain types used across handlers and
// the Supabase client. They map 1:1 onto the columns declared in
// migrations/001_init.sql so that JSON tags can be reused for both
// PostgREST round-trips and template rendering.
package models

// Profile mirrors the `profiles` table (an extension of auth.users).
type Profile struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name,omitempty"`
	Phone     string `json:"phone,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// Program is a single show in the radio schedule.
type Program struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Schedule    string    `json:"schedule"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   string `json:"created_at,omitempty"`

	// Computed / joined fields filled in by the handler before rendering.
	UpVotes     int     `json:"up_votes,omitempty"`
	DownVotes   int     `json:"down_votes,omitempty"`
	AvgRating   float64 `json:"avg_rating,omitempty"`
	RatingCount int     `json:"rating_count,omitempty"`
}

// Vote is a single +1 / -1 cast by a user on a piece of content.
type Vote struct {
	ID          string `json:"id,omitempty"`
	UserID      string `json:"user_id"`
	ContentID   string `json:"content_id"`
	ContentType string `json:"content_type"` // "program" or "music"
	Value       int    `json:"value"`        // 1 or -1
}

// Rating is a 1..5 star score for a program.
type Rating struct {
	ID        string `json:"id,omitempty"`
	UserID    string `json:"user_id"`
	ProgramID string `json:"program_id"`
	Score     int    `json:"score"`
}

// ContactRequest is an inbound CRM form submission.
type ContactRequest struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone,omitempty"`
	Address     string `json:"address,omitempty"`
	Message     string `json:"message,omitempty"`
	RequestType string `json:"request_type,omitempty"`
	Status      string `json:"status,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// VoteTotal mirrors the `vote_totals` SQL view.
type VoteTotal struct {
	ContentID   string `json:"content_id"`
	ContentType string `json:"content_type"`
	UpVotes     int    `json:"up_votes"`
	DownVotes   int    `json:"down_votes"`
	Score       int    `json:"score"`
}

// ProgramRatingAvg mirrors the `program_rating_avg` SQL view.
type ProgramRatingAvg struct {
	ProgramID   string  `json:"program_id"`
	AvgScore    float64 `json:"avg_score"`
	RatingCount int     `json:"rating_count"`
}

// MyRating is one row from /ratings with the embedded program join
// (PostgREST resolves `programs(...)` via the existing FK constraint).
type MyRating struct {
	Score     int    `json:"score"`
	CreatedAt string `json:"created_at"`
	Programs  struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		ImageURL string `json:"image_url"`
	} `json:"programs"`
}

// MyVote is a vote row enriched with the program's title/image — votes
// have no FK to programs (content can also be 'music'), so the title is
// resolved by a separate query and stitched in the handler.
type MyVote struct {
	Value        int    `json:"value"`
	ContentID    string `json:"content_id"`
	ContentType  string `json:"content_type"`
	CreatedAt    string `json:"created_at"`
	ProgramTitle string `json:"-"`
	ProgramImage string `json:"-"`
}

// Session is the value we sign and store in the user's cookie.
type Session struct {
	UserID      string `json:"uid"`
	Email       string `json:"email"`
	AccessToken string `json:"at"`
}
