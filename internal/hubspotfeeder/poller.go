package hubspotfeeder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Poller interface {
	Poll(context.Context) error
}

type GetTagsResponse struct {
	Total   int    `json:"total,omitempty"`
	Results []*Tag `json:"results,omitempty"`
}

type GetPostsResponse struct {
	Total   int     `json:"total,omitempty"`
	Results []*Post `json:"results,omitempty"`
}

type hubspotPoller struct {
	apiKey     string
	logger     *slog.Logger
	client     *http.Client
	repository PostRepository
	interval   time.Duration
}

func NewPoller(apiKey string, logger *slog.Logger, repository PostRepository, interval time.Duration) Poller {
	return &hubspotPoller{
		apiKey:     apiKey,
		logger:     logger,
		client:     &http.Client{},
		repository: repository,
		interval:   interval,
	}
}

func (p *hubspotPoller) Poll(ctx context.Context) error {
	p.logger.Info("polling started")
	defer p.logger.Info("polling stopped")

	if err := p.getTags(); err != nil {
		return fmt.Errorf("error getting tags from the host: %w", err)
	}

	if err := p.getPosts(); err != nil {
		p.logger.Error("error fetching posts", "err", err)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.getPosts(); err != nil {
				p.logger.Error("error fetching posts", "err", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *hubspotPoller) getTags() error {
	p.logger.Debug("getting tags")
	defer p.logger.Debug("tags retrieved")

	req, err := p.prepareRequest(http.MethodGet, "/blogs/tags", nil)
	if err != nil {
		return err
	}

	resp, err := p.executeRequest(req, []int{http.StatusOK})
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	var tagResponse GetTagsResponse

	if err := json.NewDecoder(resp.Body).Decode(&tagResponse); err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if err := p.repository.SetTags(tagResponse.Results); err != nil {
		return fmt.Errorf("error storing tags: %w", err)
	}

	return nil
}

func (p *hubspotPoller) prepareRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(
		method,
		fmt.Sprintf("https://api.hubapi.com/cms/v3%s", path),
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	return req, err
}

func (p *hubspotPoller) executeRequest(req *http.Request, expectedCodes []int) (*http.Response, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}

	for _, code := range expectedCodes {
		if resp.StatusCode == code {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

func (p *hubspotPoller) getPosts() error {
	p.logger.Debug("getting posts")
	defer p.logger.Debug("posts retrieved")

	for _, tag := range p.repository.GetTags() {
		fmt.Printf("Tag: %s, %s\n", tag.ID, tag.Name)
		req, err := p.prepareRequest(
			http.MethodGet,
			fmt.Sprintf("/blogs/posts?tagId__in=%s&state=PUBLISHED", tag.ID),
			nil,
		)
		if err != nil {
			return err
		}

		resp, err := p.executeRequest(req, []int{http.StatusOK})
		if err != nil {
			return fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		var postsResponse GetPostsResponse

		if err := json.NewDecoder(resp.Body).Decode(&postsResponse); err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}

		if err := p.repository.SetPostsForTag(tag.Name, postsResponse.Results); err != nil {
			return fmt.Errorf("error storing posts: %w", err)
		}
	}
	return nil
}
