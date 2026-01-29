package handler

import "net/http"

const (
	cacheStatusHeader = "X-Cache"
	cacheStatusHit    = "HIT"
	cacheStatusMiss   = "MISS"
)

func setCacheStatusHeader(w http.ResponseWriter, hit bool) {
	if hit {
		w.Header().Set(cacheStatusHeader, cacheStatusHit)
		return
	}
	w.Header().Set(cacheStatusHeader, cacheStatusMiss)
}
