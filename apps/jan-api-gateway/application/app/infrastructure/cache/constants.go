package cache

const (
	// CacheVersion is the API version prefix for cache keys.
	CacheVersion = "v1"

	// UserByPublicIDKey is the cache key template for user lookups by public ID.
	UserByPublicIDKey = CacheVersion + ":user:public_id:%s"
)
