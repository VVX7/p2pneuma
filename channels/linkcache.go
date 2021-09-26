package channels

import (
	"github.com/preludeorg/pneuma/util"
	"time"
)

// InitLinkCacheManager goroutine manages read/write ops on the Link Cache.
func InitLinkCacheManager() {
	cache := make(map[string]util.CachedLink)

	for {
		// Reads instructions of specified state from cache.
		op := <-CacheOpsChannel
		switch {
		//
		case op.Type == "read":
			op.ResponseLinks <- cache
		//
		case op.Type == "write":
			timestamp := time.Now()
			cache[op.Link] = util.CachedLink{
				State: op.State,
				Sent:  op.Sent,
				Time:  timestamp,
			}
			op.ResponseStatus <- true
		case op.Type == "trim":
			TrimLinkCache(&cache)
		default:
		}
	}
}

func UpdateSentLinks(envelope *util.Envelope) {
	c := ReadCacheLinks()
	for _, link := range envelope.Beacon.Links {
		if l, b := c[link.ID]; b {
			if l.State == "complete" {
				if l.Sent == false {
					_ = WriteCacheLink("complete", true, link.ID)
				}
			}
		}
	}
}
