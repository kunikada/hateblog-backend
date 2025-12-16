package hatena

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseRSS(t *testing.T) {
	feed, err := parseRSS(strings.NewReader(sampleRSS))
	require.NoError(t, err)
	require.Equal(t, "Hatena Hot", feed.Title)
	require.Len(t, feed.Entries, 2)

	first := feed.Entries[0]
	require.Equal(t, "Sample Title", first.Title)
	require.Equal(t, "https://example.com/a", first.URL)
	require.Equal(t, 42, first.BookmarkCount)
	require.Equal(t, []string{"Go", "Web"}, first.Subjects)
	require.Equal(t, "desc", first.Excerpt)
	require.Equal(t, "<p>Hello</p>", first.Content)
	require.Equal(t, time.Date(2025, 1, 1, 3, 4, 5, 0, time.UTC), first.PublishedAt)

	second := feed.Entries[1]
	require.Equal(t, 7, second.BookmarkCount)
	require.Equal(t, "https://example.com/b", second.URL)
}

func TestParseRSSInvalidDate(t *testing.T) {
	_, err := parseRSS(strings.NewReader(strings.ReplaceAll(sampleRSS, "2025-01-01T12:04:05+09:00", "invalid-date")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid dc:date")
}

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF
 xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
 xmlns="http://purl.org/rss/1.0/"
 xmlns:content="http://purl.org/rss/1.0/modules/content/"
 xmlns:dc="http://purl.org/dc/elements/1.1/"
 xmlns:hatena="http://www.hatena.ne.jp/info/xmlns#">
 <channel rdf:about="https://example.com/feed">
  <title>Hatena Hot</title>
  <link>https://example.com/feed</link>
  <description>desc</description>
  <items>
   <rdf:Seq>
    <rdf:li rdf:resource="https://example.com/a"/>
    <rdf:li rdf:resource="https://example.com/b"/>
   </rdf:Seq>
  </items>
 </channel>
 <item rdf:about="https://example.com/a">
  <title>Sample Title</title>
  <link>https://example.com/a</link>
  <description>desc</description>
  <content:encoded><![CDATA[<p>Hello</p>]]></content:encoded>
  <dc:date>2025-01-01T12:04:05+09:00</dc:date>
  <dc:subject>Go</dc:subject>
  <dc:subject>Web</dc:subject>
  <dc:subject>Go</dc:subject>
  <hatena:bookmarkcount>42</hatena:bookmarkcount>
 </item>
 <item rdf:about="https://example.com/b">
  <title>Second</title>
  <link>https://example.com/b</link>
  <description>second</description>
  <dc:date>2025-01-03T00:04:05Z</dc:date>
  <hatena:bookmarkcount>7</hatena:bookmarkcount>
 </item>
</rdf:RDF>`
