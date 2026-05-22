// RadioApp HTTP server entry point.
//
// Wires together the Supabase client, the Echo HTTP router, the HTML
// template renderer, and the per-route handlers defined in
// internal/handlers.
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/radioapp/radioapp/internal/handlers"
	"github.com/radioapp/radioapp/internal/supabase"
)

// templateRenderer is the small adapter Echo expects so c.Render works
// with the standard library html/template package.
//
// Each page is parsed in its own *template.Template set together with
// base.html. That lets every page override the same {{define "content"}}
// block without colliding with sibling pages — the classic stdlib trick
// for "template inheritance".
type templateRenderer struct {
	pages map[string]*template.Template
}

func (r *templateRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	t, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}
	return t.ExecuteTemplate(w, "base", data)
}

// loadTemplates builds a renderer with one template set per page.
func loadTemplates(root string) (*templateRenderer, error) {
	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"seq": func(n int) []int {
			out := make([]int, n)
			for i := range out {
				out[i] = i + 1
			}
			return out
		},
	}

	base := filepath.Join(root, "base.html")
	matches, err := filepath.Glob(filepath.Join(root, "*.html"))
	if err != nil {
		return nil, err
	}

	r := &templateRenderer{pages: map[string]*template.Template{}}
	for _, p := range matches {
		name := filepath.Base(p)
		if name == "base.html" {
			continue
		}
		t, err := template.New(name).Funcs(funcs).ParseFiles(base, p)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		r.pages[name] = t
	}
	return r, nil
}

// mustEnv returns the value of an env var or fatally fails if it is empty.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func main() {
	// Load .env if present (silent on absence so Docker / k8s deploys
	// that inject env vars directly still work).
	_ = godotenv.Load()

	supabaseURL := mustEnv("SUPABASE_URL")
	anonKey := mustEnv("SUPABASE_ANON_KEY")
	secretKey := mustEnv("SUPABASE_SECRET_KEY")
	streamURL := mustEnv("STREAM_URL")

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Println("warning: SESSION_SECRET not set, using insecure default")
		sessionSecret = "dev-insecure-please-change"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sb := supabase.New(supabaseURL, anonKey, secretKey)
	h := &handlers.Handlers{
		SB:            sb,
		SessionSecret: []byte(sessionSecret),
		StreamURL:     streamURL,
		StationName:   "Triple J – Live from Australia",
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// Templates + static files.
	r, err := loadTemplates("web/templates")
	if err != nil {
		log.Fatalf("template parse: %v", err)
	}
	e.Renderer = r
	e.Static("/static", "web/static")

	// Public routes.
	e.GET("/", h.Home)
	e.GET("/programs", h.ListPrograms)
	e.POST("/contact", h.SubmitContact)

	e.GET("/login", h.LoginPage)
	e.POST("/login", h.Login)
	e.GET("/register", h.RegisterPage)
	e.POST("/register", h.Register)
	e.POST("/logout", h.Logout)

	// Health probe for compose / k8s.
	e.GET("/healthz", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	// Protected routes (HTMX endpoints).
	auth := e.Group("", h.RequireAuth)
	auth.POST("/vote", h.CastVote)
	auth.POST("/rate", h.RateProgram)

	log.Printf("RadioApp listening on :%s", port)
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
