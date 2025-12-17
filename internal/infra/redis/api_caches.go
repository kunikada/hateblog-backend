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

func NewDayEntriesCache(client bytesCacheClient) *DayEntriesCache {
	return &DayEntriesCache{cache: newSnappyJSONCache(client, entriesDayTTL)}
}

func (c *DayEntriesCache) key(date string) string {
	return fmt.Sprintf("hateblog:entries:%s:all", date)
}

func (c *DayEntriesCache) Get(ctx context.Context, date string) ([]*domainEntry.Entry, bool, error) {
	var out []*domainEntry.Entry
	ok, err := c.cache.Get(ctx, c.key(date), &out)
	return out, ok, err
}

func (c *DayEntriesCache) Set(ctx context.Context, date string, entries []*domainEntry.Entry) error {
	return c.cache.Set(ctx, c.key(date), entries)
}

// TagEntriesCache caches all entries for a given tag.
type TagEntriesCache struct {
	cache *snappyJSONCache
}

func NewTagEntriesCache(client bytesCacheClient) *TagEntriesCache {
	return &TagEntriesCache{cache: newSnappyJSONCache(client, tagEntriesTTL)}
}

func (c *TagEntriesCache) key(tagName string) string {
	norm := domainTag.NormalizeName(tagName)
	return fmt.Sprintf("hateblog:tags:%s:entries:all", url.QueryEscape(norm))
}

func (c *TagEntriesCache) Get(ctx context.Context, tagName string) ([]*domainEntry.Entry, bool, error) {
	var out []*domainEntry.Entry
	ok, err := c.cache.Get(ctx, c.key(tagName), &out)
	return out, ok, err
}

func (c *TagEntriesCache) Set(ctx context.Context, tagName string, entries []*domainEntry.Entry) error {
	return c.cache.Set(ctx, c.key(tagName), entries)
}

// SearchCache caches full search responses for a given query+params.
type SearchCache struct {
	cache *snappyJSONCache
}

func NewSearchCache(client bytesCacheClient) *SearchCache {
	return &SearchCache{cache: newSnappyJSONCache(client, searchTTL)}
}

func (c *SearchCache) key(query string, minUsers, limit, offset int) string {
	hash := sha256Hex(strings.TrimSpace(query))
	return "hateblog:search:" + hash + ":" + strconv.Itoa(minUsers) + ":" + strconv.Itoa(limit) + ":" + strconv.Itoa(offset)
}

func (c *SearchCache) Get(ctx context.Context, query string, minUsers, limit, offset int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(query, minUsers, limit, offset), out)
}

func (c *SearchCache) Set(ctx context.Context, query string, minUsers, limit, offset int, value any) error {
	return c.cache.Set(ctx, c.key(query, minUsers, limit, offset), value)
}

// TagsListCache caches tag list responses.
type TagsListCache struct {
	cache *snappyJSONCache
}

func NewTagsListCache(client bytesCacheClient) *TagsListCache {
	return &TagsListCache{cache: newSnappyJSONCache(client, tagsListTTL)}
}

func (c *TagsListCache) key(limit, offset int) string {
	return fmt.Sprintf("hateblog:tags:list:%d:%d", limit, offset)
}

func (c *TagsListCache) Get(ctx context.Context, limit, offset int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(limit, offset), out)
}

func (c *TagsListCache) Set(ctx context.Context, limit, offset int, value any) error {
	return c.cache.Set(ctx, c.key(limit, offset), value)
}

// ArchiveCache caches archive responses.
type ArchiveCache struct {
	cache *snappyJSONCache
}

func NewArchiveCache(client bytesCacheClient) *ArchiveCache {
	return &ArchiveCache{cache: newSnappyJSONCache(client, archiveTTL)}
}

func (c *ArchiveCache) key(minUsers int) string {
	return fmt.Sprintf("hateblog:archive:all:%d", minUsers)
}

func (c *ArchiveCache) Get(ctx context.Context, minUsers int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(minUsers), out)
}

func (c *ArchiveCache) Set(ctx context.Context, minUsers int, value any) error {
	return c.cache.Set(ctx, c.key(minUsers), value)
}

// YearlyRankingCache caches yearly ranking entries (up to max) per min_users.
type YearlyRankingCache struct {
	client bytesCacheClient
}

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

func (c *YearlyRankingCache) Get(ctx context.Context, year, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, time.Now())).Get(ctx, c.key(year, minUsers), out)
}

func (c *YearlyRankingCache) Set(ctx context.Context, year, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, time.Now())).Set(ctx, c.key(year, minUsers), value)
}

// MonthlyRankingCache caches monthly ranking entries (up to max) per min_users.
type MonthlyRankingCache struct {
	client bytesCacheClient
}

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

func (c *MonthlyRankingCache) Get(ctx context.Context, year, month, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, month, time.Now())).Get(ctx, c.key(year, month, minUsers), out)
}

func (c *MonthlyRankingCache) Set(ctx context.Context, year, month, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, month, time.Now())).Set(ctx, c.key(year, month, minUsers), value)
}

// WeeklyRankingCache caches weekly ranking entries (up to max) per min_users.
type WeeklyRankingCache struct {
	client bytesCacheClient
}

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

func (c *WeeklyRankingCache) Get(ctx context.Context, year, week, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Get(ctx, c.key(year, week, minUsers), out)
}

func (c *WeeklyRankingCache) Set(ctx context.Context, year, week, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Set(ctx, c.key(year, week, minUsers), value)
}
