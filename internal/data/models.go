package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound is returned when the specified item is not found.
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Models contains all the models that are declared and will be passed around as dependency.
type Models struct {
	Words       WordModel
	Tokens      TokenModel
	Users       UserModel
	Permissions PermissionModel
}

// NewModels returns an initialised Models to everything.
func NewModels(db *sql.DB) Models {
	return Models{
		Words:       WordModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Users:       UserModel{DB: db},
		Permissions: PermissionModel{DB: db},
	}
}
