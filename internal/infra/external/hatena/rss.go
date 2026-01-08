package hatena

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"hateblog/internal/pkg/timeutil"
)

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
	Items   []rssItem  `xml:"item"`
}

type rssChannel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type rssItem struct {
	Title         string   `xml:"title"`
	Link          string   `xml:"link"`
	Description   string   `xml:"description"`
	Content       string   `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Date          string   `xml:"http://purl.org/dc/elements/1.1/ date"`
	Subjects      []string `xml:"http://purl.org/dc/elements/1.1/ subject"`
	BookmarkCount string   `xml:"http://www.hatena.ne.jp/info/xmlns# bookmarkcount"`
}

func parseRSS(r io.Reader) (*Feed, error) {
	var doc rssDocument
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("hatena: failed to parse rss: %w", err)
	}

	feed := &Feed{
		Title:       strings.TrimSpace(doc.Channel.Title),
		Description: strings.TrimSpace(doc.Channel.Description),
		Link:        strings.TrimSpace(doc.Channel.Link),
	}

	for _, item := range doc.Items {
		publishedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Date))
		if err != nil {
			return nil, fmt.Errorf("hatena: invalid dc:date %q: %w", item.Date, err)
		}
		publishedAt = timeutil.InLocation(publishedAt)

		count := 0
		if strings.TrimSpace(item.BookmarkCount) != "" {
			count, err = strconv.Atoi(strings.TrimSpace(item.BookmarkCount))
			if err != nil {
				return nil, fmt.Errorf("hatena: invalid bookmark count %q: %w", item.BookmarkCount, err)
			}
		}

		entry := FeedEntry{
			Title:         strings.TrimSpace(item.Title),
			URL:           strings.TrimSpace(item.Link),
			Excerpt:       strings.TrimSpace(item.Description),
			Content:       strings.TrimSpace(item.Content),
			Subjects:      normalizeSubjects(item.Subjects),
			BookmarkCount: count,
			PublishedAt:   publishedAt,
		}
		feed.Entries = append(feed.Entries, entry)
	}

	return feed, nil
}

func normalizeSubjects(subjects []string) []string {
	if len(subjects) == 0 {
		return nil
	}
	result := make([]string, 0, len(subjects))
	seen := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		if _, ok := seen[subject]; ok {
			continue
		}
		seen[subject] = struct{}{}
		result = append(result, subject)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
