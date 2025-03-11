package hubspotfeeder

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func MakeHttpHandler(repository PostRepository) http.Handler {
	m := chi.NewMux()

	m.Handle("GET /news/{tag}/rss", makeRssHandler(repository))

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
