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

// CacheTTLConfig holds TTL settings for API caches.
type CacheTTLConfig struct {
	EntriesDayTTL            time.Duration
	TagEntriesTTL            time.Duration
	SearchTTL                time.Duration
	TagsListTTL              time.Duration
	ArchiveTTL               time.Duration
	YearlyRankingCurrentTTL  time.Duration
	YearlyRankingPastTTL     time.Duration
	MonthlyRankingCurrentTTL time.Duration
	MonthlyRankingPastTTL    time.Duration
	WeeklyRankingCurrentTTL  time.Duration
	WeeklyRankingPastTTL     time.Duration
}

// DayEntriesCache caches all entries for a given JST date (YYYYMMDD).
type DayEntriesCache struct {
	cache *snappyJSONCache
}

// NewDayEntriesCache builds a day entries cache.
func NewDayEntriesCache(client bytesCacheClient, ttl time.Duration) *DayEntriesCache {
	return &DayEntriesCache{cache: newSnappyJSONCache(client, ttl)}
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

// TagEntriesCache caches the first page of tag entries for a given tag.
type TagEntriesCache struct {
	cache *snappyJSONCache
}

// NewTagEntriesCache builds a tag entries cache.
func NewTagEntriesCache(client bytesCacheClient, ttl time.Duration) *TagEntriesCache {
	return &TagEntriesCache{cache: newSnappyJSONCache(client, ttl)}
}

func (c *TagEntriesCache) key(tagName string, sort domainEntry.SortType, minUsers int) string {
	norm := domainTag.NormalizeName(tagName)
	return fmt.Sprintf("hateblog:tags:%s:entries:%s:%d:100:0", url.QueryEscape(norm), sort, minUsers)
}

// Get returns cached tag entries for the given tag name.
func (c *TagEntriesCache) Get(ctx context.Context, tagName string, sort domainEntry.SortType, minUsers int, out any) (bool, error) {
	return c.cache.Get(ctx, c.key(tagName, sort, minUsers), out)
}

// Set stores tag entries for the given tag name.
func (c *TagEntriesCache) Set(ctx context.Context, tagName string, sort domainEntry.SortType, minUsers int, value any) error {
	return c.cache.Set(ctx, c.key(tagName, sort, minUsers), value)
}

// SearchCache caches full search responses for a given query+params.
type SearchCache struct {
	cache *snappyJSONCache
}

// NewSearchCache builds a search cache.
func NewSearchCache(client bytesCacheClient, ttl time.Duration) *SearchCache {
	return &SearchCache{cache: newSnappyJSONCache(client, ttl)}
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
func NewTagsListCache(client bytesCacheClient, ttl time.Duration) *TagsListCache {
	return &TagsListCache{cache: newSnappyJSONCache(client, ttl)}
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

// ArchiveCache caches archive responses with separate TTLs for today and past data.
type ArchiveCache struct {
	client   bytesCacheClient
	todayTTL time.Duration
	pastTTL  time.Duration
}

// NewArchiveCache builds an archive cache.
// todayTTL is used for today's data, pastTTL for historical data.
func NewArchiveCache(client bytesCacheClient, todayTTL, pastTTL time.Duration) *ArchiveCache {
	return &ArchiveCache{client: client, todayTTL: todayTTL, pastTTL: pastTTL}
}

func (c *ArchiveCache) todayKey(minUsers int) string {
	return fmt.Sprintf("hateblog:archive:today:%d", minUsers)
}

func (c *ArchiveCache) pastKey(minUsers int) string {
	return fmt.Sprintf("hateblog:archive:past:%d", minUsers)
}

// GetToday returns cached today's archive count.
func (c *ArchiveCache) GetToday(ctx context.Context, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.todayTTL).Get(ctx, c.todayKey(minUsers), out)
}

// SetToday stores today's archive count.
func (c *ArchiveCache) SetToday(ctx context.Context, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.todayTTL).Set(ctx, c.todayKey(minUsers), value)
}

// GetPast returns cached past archive counts.
func (c *ArchiveCache) GetPast(ctx context.Context, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.pastTTL).Get(ctx, c.pastKey(minUsers), out)
}

// SetPast stores past archive counts.
func (c *ArchiveCache) SetPast(ctx context.Context, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.pastTTL).Set(ctx, c.pastKey(minUsers), value)
}

// YearlyRankingCache caches yearly ranking entries (up to max) per min_users.
type YearlyRankingCache struct {
	client     bytesCacheClient
	currentTTL time.Duration
	pastTTL    time.Duration
}

// NewYearlyRankingCache builds a yearly ranking cache.
func NewYearlyRankingCache(client bytesCacheClient, currentTTL, pastTTL time.Duration) *YearlyRankingCache {
	return &YearlyRankingCache{client: client, currentTTL: currentTTL, pastTTL: pastTTL}
}

func (c *YearlyRankingCache) key(year, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:yearly:%d:%d", year, minUsers)
}

func (c *YearlyRankingCache) ttl(year int, now time.Time) time.Duration {
	if year == now.Year() {
		return c.currentTTL
	}
	return c.pastTTL
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
	client     bytesCacheClient
	currentTTL time.Duration
	pastTTL    time.Duration
}

// NewMonthlyRankingCache builds a monthly ranking cache.
func NewMonthlyRankingCache(client bytesCacheClient, currentTTL, pastTTL time.Duration) *MonthlyRankingCache {
	return &MonthlyRankingCache{client: client, currentTTL: currentTTL, pastTTL: pastTTL}
}

func (c *MonthlyRankingCache) key(year, month, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:monthly:%d:%d:%d", year, month, minUsers)
}

func (c *MonthlyRankingCache) ttl(year, month int, now time.Time) time.Duration {
	if year == now.Year() && month == int(now.Month()) {
		return c.currentTTL
	}
	return c.pastTTL
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
	client     bytesCacheClient
	currentTTL time.Duration
	pastTTL    time.Duration
}

// NewWeeklyRankingCache builds a weekly ranking cache.
func NewWeeklyRankingCache(client bytesCacheClient, currentTTL, pastTTL time.Duration) *WeeklyRankingCache {
	return &WeeklyRankingCache{client: client, currentTTL: currentTTL, pastTTL: pastTTL}
}

func (c *WeeklyRankingCache) key(year, week, minUsers int) string {
	return fmt.Sprintf("hateblog:rankings:weekly:%d:%d:%d", year, week, minUsers)
}

func (c *WeeklyRankingCache) ttl(year, week int, now time.Time) time.Duration {
	nowYear, nowWeek := now.ISOWeek()
	if year == nowYear && week == nowWeek {
		return c.currentTTL
	}
	return c.pastTTL
}

// Get returns cached weekly rankings.
func (c *WeeklyRankingCache) Get(ctx context.Context, year, week, minUsers int, out any) (bool, error) {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Get(ctx, c.key(year, week, minUsers), out)
}

// Set stores weekly rankings.
func (c *WeeklyRankingCache) Set(ctx context.Context, year, week, minUsers int, value any) error {
	return newSnappyJSONCache(c.client, c.ttl(year, week, time.Now())).Set(ctx, c.key(year, week, minUsers), value)
}
