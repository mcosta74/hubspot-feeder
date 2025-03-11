package hubspotfeeder

import (
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("item not found")
)

type Post struct {
	ID          string    `json:"id,omitempty"`
	PublishDate time.Time `json:"publishDate,omitempty"`
	Meta        string    `json:"metaDescription,omitempty"`
	Url         string    `json:"url,omitempty"`
	Summary     string    `json:"postSummary,omitempty"`
	Name        string    `json:"name,omitempty"`
	Title       string    `json:"htmlTitle,omitempty"`
}

type Tag struct {
	ID        string    `json:"id,omitempty"`
	Name      string    `json:"name,omitempty"`
	Created   time.Time `json:"created,omitempty"`
	DeletedAt time.Time `json:"deletedAt,omitempty"`
	Updated   time.Time `json:"updated,omitempty"`
}

type PostRepository interface {
	SetTags(tags []*Tag) error
	GetTags() []*Tag

	GetPostsForTag(tag string) ([]*Post, error)
	SetPostsForTag(tag string, posts []*Post) error
}

func NewRepository() PostRepository {
	return &inmemRepository{
		tags:  make(map[string]*Tag),
		posts: make(map[string][]*Post),
	}
}

type inmemRepository struct {
	tagMutex sync.Mutex
	tags     map[string]*Tag

	postMutex sync.RWMutex
	posts     map[string][]*Post
}

// Check interface implementations at compile time
var (
	_ PostRepository = &inmemRepository{}
)

func (r *inmemRepository) GetPostsForTag(tag string) ([]*Post, error) {
	r.postMutex.RLock()
	defer r.postMutex.RUnlock()

	posts, ok := r.posts[tag]
	if !ok {
		return nil, ErrNotFound
	}
	return posts, nil
}

func (r *inmemRepository) SetPostsForTag(tag string, posts []*Post) error {
	r.postMutex.Lock()
	defer r.postMutex.Unlock()

	r.posts[tag] = posts

	return nil
}

func (r *inmemRepository) SetTags(tags []*Tag) error {
	r.tagMutex.Lock()
	defer r.tagMutex.Unlock()

	clear(r.tags)

	for _, tag := range tags {
		// add only not deleted tags
		if tag.DeletedAt.Unix() == 0 {
			r.tags[tag.ID] = tag
		}
	}
	return nil
}

func (r *inmemRepository) GetTags() []*Tag {
	return slices.Collect(maps.Values(r.tags))
}
