package hubspotfeeder

import (
	"encoding/xml"
	"fmt"
	"io"
	"text/template"
	"time"
)

// Define the Atom feed structure.
type AtomFeed struct {
	Title   string
	ID      string
	Updated time.Time
	Entries []AtomEntry
}

type AtomEntry struct {
	ID      string
	Title   string
	Updated time.Time
	Link    string
	Meta    string
}

func writeFeed(w io.Writer, tag string, posts []*Post) error {

	templateStr := xml.Header +
		`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:fw="http://example.com/xml/fw">
	<title>{{ .Title }}</title>
	<id>{{ .ID }}</id>
	<updated>{{ .Updated.Format "2006-01-02T15:04:05Z07:00"}}</updated>
  	{{- range .Entries }}	
	<entry>
		<id>{{ .ID }}</id>
		<title>{{ .Title }}</title>
		<updated>{{ .Updated.Format "2006-01-02T15:04:05Z07:00"}}</updated>
		<link rel="alternate" type="text/html" href="{{ .Link }}"/>
		<fw:metadata>{{ .Meta }}<fw:metadata>
	</entry>
	{{- end}}
</feed>
	`

	tmpl, err := template.New("atom").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	feed := AtomFeed{
		Title:   fmt.Sprintf("Tag:%s", tag),
		ID:      fmt.Sprintf("fw:%s", tag),
		Updated: time.Now(),
	}
	for _, post := range posts {
		feed.Entries = append(feed.Entries, AtomEntry{
			ID:      post.ID,
			Title:   post.Title,
			Updated: post.PublishDate,
			Link:    post.Url,
			Meta:    post.Meta,
		})
	}
	err = tmpl.Execute(w, feed)
	if err != nil {
		return fmt.Errorf("error building template: %w", err)
	}
	return nil
}
