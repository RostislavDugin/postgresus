package stream_guard

import "errors"

var ErrDownloadAlreadyInProgress = errors.New("download already in progress for this user")
