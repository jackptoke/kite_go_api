package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"kite-api/internal/validator"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Word struct {
	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"-"`
	TextValue    string    `json:"text"`
	Difficulty   string    `json:"difficulty"`
	RelatedWords []string  `json:"related_words,omitempty"`
	UserId       int64     `json:"user_id,omitempty"`
	Version      int32     `json:"-"`
}

// ValidateWord checks if all its fields are provided with valid values
func ValidateWord(v *validator.Validator, word *Word) {
	v.Check(strings.Trim(word.TextValue, " ") != "", "text", "word text must be provided")
	v.Check(len(word.TextValue) <= 255, "title", "word text must be 255 or less characters")

	difficulties := []string{"easy", "medium", "hard"}

	v.Check(includes(difficulties, word.Difficulty), "difficulty", "difficulty must be one of: easy, medium, hard")

	v.Check(word.UserId > 0, "user_id", "user_id must be a valid user id number")
}

// MarshalJSON converts the Go Word object to a JSON value
func (w Word) MarshalJSON() ([]byte, error) {
	var userId string

	if w.UserId != 0 {
		userId = fmt.Sprintf("ID%d", w.UserId)
	}

	type WordAlias Word

	aux := struct {
		WordAlias
		UserId string `json:"user_id,omitempty"`
	}{
		WordAlias: WordAlias(w),
		UserId:    userId,
	}
	return json.Marshal(aux)
}

func includes(values []string, v string) bool {
	for _, w := range values {
		if w == v {
			return true
		}
	}
	return false
}

type WordModel struct {
	DB *sql.DB
}

// Insert a new entry of word or text
func (w WordModel) Insert(word *Word) error {

	query := `INSERT INTO words (text_value, difficulty, related_words, user_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{word.TextValue, word.Difficulty, pq.Array(word.RelatedWords), word.UserId}
	return w.DB.QueryRowContext(ctx, query, args...).Scan(&word.ID, &word.CreatedAt)
}

// Get retrieves a record that matches the id
func (w WordModel) Get(id int64) (*Word, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `SELECT id, text_value, difficulty, related_words, user_id, created_at, version FROM words WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var word Word
	err := w.DB.QueryRowContext(ctx, query, id).Scan(&word.ID, &word.TextValue, &word.Difficulty, pq.Array(&word.RelatedWords), &word.UserId, &word.CreatedAt, &word.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &word, nil
}

// Update the word
func (w WordModel) Update(word *Word) error {
	query := `UPDATE words SET text_value = $1, difficulty = $2, related_words = $3, user_id = $4, version = version + 1 WHERE id = $5 AND version = $6 RETURNING version`
	args := []any{word.TextValue, word.Difficulty, pq.Array(word.RelatedWords), word.UserId, word.ID, word.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := w.DB.QueryRowContext(ctx, query, args...).Scan(&word.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}

// Delete the word that matches the id and text
func (w WordModel) Delete(id int64) error {
	// this is probably an overkill, but I want to be sure that we are deleting the correct word
	query := `DELETE FROM words WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := w.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// GetAll
func (w WordModel) GetAll(text string, difficulty string, filters Filters) ([]*Word, Metadata, error) {
	// to_tsvector extracts the words out and disregards all the rest like punctuation and symbols.
	// plainto_tsquery extracts words out from the search terms.
	// @@ matches the left side and the right side.
	query := fmt.Sprintf(`SELECT count(*) OVER(), id, text_value, difficulty, related_words, user_id, created_at, version 
FROM words WHERE (to_tsvector('simple', text_value) @@ plainto_tsquery('simple', $1) OR $1 = '') AND (LOWER(difficulty) = LOWER($2) OR $2 = '')
ORDER BY %s %s, id ASC LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{text, difficulty, filters.limit(), filters.offset()}

	rows, err := w.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	var words []*Word

	totalRecords := 0
	for rows.Next() {
		var word Word
		err := rows.Scan(
			&totalRecords,
			&word.ID,
			&word.TextValue,
			&word.Difficulty,
			pq.Array(&word.RelatedWords),
			&word.UserId,
			&word.CreatedAt,
			&word.Version)
		if err != nil {
			return nil, Metadata{}, err
		}
		words = append(words, &word)
		totalRecords++
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return words, metadata, nil
}
