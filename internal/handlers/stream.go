package handlers

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// GET /stream.mp3  and  GET /stream.ogg
//
// Relays the upstream Faith FM Icecast stream through our own origin.
// Why bother? Chrome's <audio> element is finicky with
// Transfer-Encoding: chunked over long-haul TLS (browser ↔ Australia).
// Routing it through same-origin lets Chrome use the already-warm
// Render TLS session, and the Go-to-Icecast leg is a cleaner HTTP/1.1
// client that doesn't make assumptions about Content-Length / Range.
//
// Resource cost per listener: ~12 KB/s egress for 96 kbps MP3, plus one
// goroutine and one TCP connection to upstream. Fine for a portfolio
// demo on Render's free tier (100 GB/month egress ≈ 2300 listener-hours).

var proxyClient = &http.Client{
	// No overall timeout: the stream is meant to run forever. Per-read
	// timeouts are enforced by the Transport dial / idle settings below.
	Timeout: 0,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     120 * time.Second,
		DisableCompression:  true, // audio is already compressed
	},
}

// upstreamFor picks the right env URL based on the requested extension.
func (h *Handlers) upstreamFor(path string) string {
	switch {
	case strings.HasSuffix(path, ".ogg"):
		return h.StreamURLFallback
	default:
		return h.StreamURL
	}
}

func (h *Handlers) ProxyStream(c echo.Context) error {
	upstream := h.upstreamFor(c.Request().URL.Path)
	if upstream == "" {
		return c.NoContent(http.StatusNotFound)
	}

	// Build an upstream request tied to the client's context so a
	// browser disconnect cancels the goroutine instead of leaking it.
	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet, upstream, nil)
	if err != nil {
		return c.String(http.StatusBadGateway, "bad upstream URL")
	}
	req.Header.Set("User-Agent", "RadioApp-Proxy/1.0")
	// Ask the Icecast server for inline Icy metadata so we could show
	// "now playing" later. Harmless if the server ignores it.
	req.Header.Set("Icy-MetaData", "0")

	resp, err := proxyClient.Do(req)
	if err != nil {
		return c.String(http.StatusBadGateway, "upstream unreachable: "+err.Error())
	}
	defer resp.Body.Close()

	// Pass through just the headers the browser actually needs. We
	// deliberately do NOT forward Transfer-Encoding — Go's HTTP server
	// re-encodes the response on its own.
	passthrough := []string{"Content-Type", "icy-br", "icy-name", "icy-description"}
	for _, k := range passthrough {
		if v := resp.Header.Get(k); v != "" {
			c.Response().Header().Set(k, v)
		}
	}
	c.Response().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response().WriteHeader(resp.StatusCode)

	// Stream bytes until either side closes. io.Copy flushes as it
	// writes because echo's ResponseWriter wraps an http.Flusher — but
	// we force a flush every chunk just in case the runtime decides to
	// buffer. Keeping chunks small (4 KB) means low latency: a listener
	// never waits more than ~340 ms for a 96 kbps MP3 slice.
	flusher, _ := c.Response().Writer.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := c.Response().Writer.Write(buf[:n]); werr != nil {
				return nil // browser went away; graceful exit
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return nil
			}
			return nil // upstream dropped — nothing to do, client will reconnect
		}
	}
}
