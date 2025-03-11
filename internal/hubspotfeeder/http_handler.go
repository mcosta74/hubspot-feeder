package hubspotfeeder

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func MakeHttpHandler(repository PostRepository, logger *slog.Logger) http.Handler {
	m := chi.NewMux()

	var rssHandler http.Handler
	{
		rssHandler = makeRssHandler(repository)
		rssHandler = makeLoggingMiddleware(logger)(rssHandler)
	}
	m.Handle("GET /news/{tag}/rss", rssHandler)

	return m
}

func makeRssHandler(repository PostRepository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tag := r.PathValue("tag")

		posts, err := repository.GetPostsForTag(tag)
		if err != nil {
			if err == ErrNotFound {
				http.NotFound(w, r)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		err = writeFeed(w, tag, posts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func makeLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := middleware.NewWrapResponseWriter(w, 0)
			defer func(dt time.Time) {
				logger.Debug("request handled", "path", r.URL.Path, "elapsed", time.Since(dt), "status_code", rw.Status(), "bytes_written", rw.BytesWritten())
			}(time.Now())

			// forward the request to the next handler
			next.ServeHTTP(rw, r)
		})
	}
}
