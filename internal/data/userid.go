package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type UserID int64

var ErrInvalidUserIdFormat = errors.New("invalid user id format")

func (i UserID) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("ID-%d", i)

	quotedJsonValue := strconv.Quote(jsonValue)

	return []byte(quotedJsonValue), nil
}

func (i *UserID) UnmarshalJSON(jsonValue []byte) error {
	// Extracting the JSON value
	unquotedJsonValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidUserIdFormat
	}

	parts := strings.Split(unquotedJsonValue, "-")

	// The ID must strictly be started with ID, follows by a dash, '-', and then the number.
	// Otherwise, we return the ErrInvalidUserIdFormat
	if len(parts) != 2 || parts[0] != "ID" {
		return ErrInvalidUserIdFormat
	}

	// Parse the ID into an int
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	// Convert the id into UserID
	*i = UserID(id)

	return nil
}
