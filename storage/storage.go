package storage

import (
	"github.com/nyxlumen/nyx-browser/v2/auth"
	"github.com/nyxlumen/nyx-browser/v2/settings"
	"github.com/nyxlumen/nyx-browser/v2/share"
	"github.com/nyxlumen/nyx-browser/v2/users"
)

// Storage is a storage powered by a Backend which makes the necessary
// verifications when fetching and saving data to ensure consistency.
type Storage struct {
	Users    users.Store
	Share    *share.Storage
	Auth     *auth.Storage
	Settings *settings.Storage
}
