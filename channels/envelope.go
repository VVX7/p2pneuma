package channels

import (
	"github.com/preludeorg/pneuma/util"
)

// InitEnvelopeManager passes Envelopes to the EnvelopeHandler.
func InitEnvelopeManager(f func(envelope *util.Envelope)) {
	for {
		envelope := <-Envelopes
		cacheLinks := ReadCacheLinks()

		// Check if each Link in the Beacon is cached.
		for _, link := range envelope.Beacon.Links {
			if util.CacheContains(cacheLinks, link.ID) {
				// If a Link is complete not sent we retry execution.
				if cacheLinks[link.ID].State == "complete" {
					if cacheLinks[link.ID].Sent == false {
						e := util.BuildSingleLinkEnvelope(envelope, link)
						f(e)
					}
				}
				// Pass it to an executor if not cached.
			} else {
				e := util.BuildSingleLinkEnvelope(envelope, link)
				// Links are marked as pending prior to being handled by the executor.
				if WriteCacheLink("pending", false, link.ID) {
					f(e)
				}
			}
		}
	}
}
