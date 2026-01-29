package redis

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	domainEntry "hateblog/internal/domain/entry"
	domainTag "hateblog/internal/domain/tag"
)

const (
	entriesDayTTL = 5 * time.Minute
	tagEntriesTTL = 10 * time.Minute
	searchTTL     = 15 * time.Minute
	tagsListTTL   = 1 * time.Hour
	archiveTTL    = 24 * time.Hour

	yearlyRankingCurrentTTL  = 1 * time.Hour
	yearlyRankingPastTTL     = 7 * 24 * time.Hour
	monthlyRankingCurrentTTL = 1 * time.Hour
	monthlyRankingPastTTL    = 24 * time.Hour
	weeklyRankingCurrentTTL  = 30 * time.Minute
	weeklyRankingPastTTL     = 24 * time.Hour
)

// DayEntriesCache caches all entries for a given JST date (YYYYMMDD).
type DayEntriesCache struct {
	cache *snappyJSONCache
}

// NewDayEntriesCache builds a day entries cache.
func NewDayEntriesCache(client bytesCacheClient) *DayEntriesCache {
	return &DayEntriesCache{cache: newSnappyJSONCache(client, entriesDayTTL)}
}

func (c *DayEntriesCache) key(date string) string {
	return fmt.Sprintf("hateblog:entries:%s:all", date)
}

// Get returns cached day entries for the given date.
func (c *DayEntriesCache) Get(ctx context.Context, date string) ([]*domainEntry.Entry, bool, error) {
	var out []*domainEntry.Entry
	ok, err := c.cache.Get(ctx, c.key(date), &out)
	return out, ok, err
}

// Set stores day entries for the given date.
func (c *DayEntriesCache) Set(ctx context.Context, date string, entries []*domainEntry.Entry) error {
	return c.cache.Set(ctx, c.key(date), entries)
}

// TagEntriesCache caches all entries for a given tag.
type TagEntriesCache struct {
	cache *snappyJSONCache
}

// NewTagEntriesCache builds a tag entries cache.
func NewTagEntriesCache(client bytesCacheClient) *TagEntriesCache {
	return &TagEntriesCache{cache: newSnappyJSONCache(client, tagEntriesTTL)}
}

func (c *TagEntriesCache) key(tagName string) string {
	norm := domainTag.NormalizeName(tagName)
	return fmt.Sprintf("hateblog:tags:%s:entries:all", url.QueryEscape(norm))
}

// Get returns cached tag entries for the given tag name.
func (c *TagEntriesCache) Get(ctx context.Context, tagName string) ([]*domainEntry.Entry, bool, error) {
	var out []*domainEntry.Entry
	ok, err := c.cache.Get(ctx, c.key(tagName), &out)
	return out, ok, err
}

// Set stores tag entries for the given tag name.
func (c *TagEntriesCache) Set(ctx context.Context, tagName string, entries []*domainEntry.Entry) error {
	return c.cache.Set(ctx, c.key(tagName), entries)
}

// SearchCache caches full search responses for a given query+params.
type SearchCache struct {
	cache *snappyJSONCache
}

// NewSearchCache builds a search cache.
func NewSearchCache(client bytesCacheClient) *SearchCache {
	return &SearchCache{cache: newSnappyJSONCache(client, searchTTL)}
}

func (c *SearchCache) key(query string, sort domainEntry.SortType, minUsers, limit, offset int) string {
	hash := sha256Hex(strings.TrimSpace(query))
	return "hateblog:search:" + hash + ":" + string(sort) + ":" + strconv.Itoa(minUsers) + ":" + strconv.Itoa(limit) + ":" + strconv.Itoa(offset)
}

// Get returns cached search results for the given query parameters.
func (c *SearchCache) Get(ctx context.Context, query string, sort domainEntry.SortType, minUsers, limit, offset int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(query, sort, minUsers, limit, offset), out)
}

// Set stores search results for the given query parameters.
func (c *SearchCache) Set(ctx context.Context, query string, sort domainEntry.SortType, minUsers, limit, offset int, value any) error {
	return c.cache.Set(ctx, c.key(query, sort, minUsers, limit, offset), value)
}

// TagsListCache caches tag list responses.
type TagsListCache struct {
	cache *snappyJSONCache
}

// NewTagsListCache builds a tags list cache.
func NewTagsListCache(client bytesCacheClient) *TagsListCache {
	return &TagsListCache{cache: newSnappyJSONCache(client, tagsListTTL)}
}

func (c *TagsListCache) key(limit, offset int) string {
	return fmt.Sprintf("hateblog:tags:list:%d:%d", limit, offset)
}

// Get returns cached tag list responses.
func (c *TagsListCache) Get(ctx context.Context, limit, offset int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(limit, offset), out)
}

// Set stores tag list responses.
func (c *TagsListCache) Set(ctx context.Context, limit, offset int, value any) error {
	return c.cache.Set(ctx, c.key(limit, offset), value)
}

// ArchiveCache caches archive responses.
type ArchiveCache struct {
	cache *snappyJSONCache
}

// NewArchiveCache builds an archive cache.
func NewArchiveCache(client bytesCacheClient) *ArchiveCache {
	return &ArchiveCache{cache: newSnappyJSONCache(client, archiveTTL)}
}

func (c *ArchiveCache) key(minUsers int) string {
	return fmt.Sprintf("hateblog:archive:all:%d", minUsers)
}

// Get returns cached archive responses.
func (c *ArchiveCache) Get(ctx context.Context, minUsers int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(minUsers), out)
}

// Set stores archive responses.
func (c *ArchiveCache) Set(ctx context.Context, minUsers int, value any) error {
	return c.cache.Set(ctx, c.key(minUsers), value)
}

// YearlyRankingCache caches yearly ranking entries (up to max) per min_users.
type YearlyRankingCache struct {
	client bytesCacheClient
}

// NewYearlyRankingCache builds a yearly ranking cache.
func NewYearlyRankingCache(client bytesCacheClient) *YearlyRankingCache {
	return &YearlyRankingCache{client: client}
}

func (c *YearlyRankingCache) key(year, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:yearly:%d:%d", year, minUsers)
}

func (c *YearlyRankingCache) ttl(year int, now time.Time) time.Duration {
	if year == now.Year() {
		return yearlyRankingCurrentTTL
	}
	return yearlyRankingPastTTL
}

// Get returns cached yearly rankings.
func (c *YearlyRankingCache) Get(ctx context.Context, year, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, time.Now())).Get(ctx, c.key(year, minUsers), out)
}

// Set stores yearly rankings.
func (c *YearlyRankingCache) Set(ctx context.Context, year, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, time.Now())).Set(ctx, c.key(year, minUsers), value)
}

// MonthlyRankingCache caches monthly ranking entries (up to max) per min_users.
type MonthlyRankingCache struct {
	client bytesCacheClient
}

// NewMonthlyRankingCache builds a monthly ranking cache.
func NewMonthlyRankingCache(client bytesCacheClient) *MonthlyRankingCache {
	return &MonthlyRankingCache{client: client}
}

func (c *MonthlyRankingCache) key(year, month, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:monthly:%d:%d:%d", year, month, minUsers)
}

func (c *MonthlyRankingCache) ttl(year, month int, now time.Time) time.Duration {
	if year == now.Year() && month == int(now.Month()) {
		return monthlyRankingCurrentTTL
	}
	return monthlyRankingPastTTL
}

// Get returns cached monthly rankings.
func (c *MonthlyRankingCache) Get(ctx context.Context, year, month, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, month, time.Now())).Get(ctx, c.key(year, month, minUsers), out)
}

// Set stores monthly rankings.
func (c *MonthlyRankingCache) Set(ctx context.Context, year, month, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, month, time.Now())).Set(ctx, c.key(year, month, minUsers), value)
}

// WeeklyRankingCache caches weekly ranking entries (up to max) per min_users.
type WeeklyRankingCache struct {
	client bytesCacheClient
}

// NewWeeklyRankingCache builds a weekly ranking cache.
func NewWeeklyRankingCache(client bytesCacheClient) *WeeklyRankingCache {
	return &WeeklyRankingCache{client: client}
}

func (c *WeeklyRankingCache) key(year, week, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:weekly:%d:%d:%d", year, week, minUsers)
}

func (c *WeeklyRankingCache) ttl(year, week int, now time.Time) time.Duration {
	nowYear, nowWeek := now.ISOWeek()
	if year == nowYear && week == nowWeek {
		return weeklyRankingCurrentTTL
	}
	return weeklyRankingPastTTL
}

// Get returns cached weekly rankings.
func (c *WeeklyRankingCache) Get(ctx context.Context, year, week, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Get(ctx, c.key(year, week, minUsers), out)
}

// Set stores weekly rankings.
func (c *WeeklyRankingCache) Set(ctx context.Context, year, week, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Set(ctx, c.key(year, week, minUsers), value)
}
