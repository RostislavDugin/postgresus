package mattermost_notifier

import "time"

const (
	// Mattermost rejects longer posts on the REST API and silently splits them on incoming webhooks.
	maxPostLength = 16383

	// Mattermost IDs are always 26 characters, which is how a pasted channel *name* is caught.
	channelIDLength = 26

	// A failed send surfaces the response body to the caller of the test-send endpoint, so a
	// misbehaving server must not be able to push an arbitrarily large body through it.
	maxErrorBodyBytes = 8 * 1024

	sendTimeout = 30 * time.Second

	createPostPath = "/api/v4/posts"
)
