package main

import (
	"errors"
	"fmt"
	"kite-api/internal/data"
	"kite-api/internal/validator"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func (app *application) createWordHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text         string      `json:"text"`
		Difficulty   string      `json:"difficulty"`
		RelatedWords []string    `json:"related_words"`
		UserId       data.UserID `json:"user_id"`
	}

	// alternative way => err := json.NewDecoder(r.Body).Decode(&input)
	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err.Error())
		return
	}

	word := &data.Word{
		ID:           rand.Int63(),
		TextValue:    input.Text,
		Difficulty:   input.Difficulty,
		RelatedWords: input.RelatedWords,
		UserId:       int64(input.UserId),
		CreatedAt:    time.Now(),
	}

	if word.RelatedWords == nil {
		word.RelatedWords = []string{}
	}

	v := validator.New()

	if data.ValidateWord(v, word); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert into the database
	err = app.models.Words.Insert(word)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Adding the url of the newly inserted record
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/api/v1/words/%d", word.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"word": &word}, headers)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getWordHandler(w http.ResponseWriter, r *http.Request) {
	// utilising the helper method
	id, err := app.readIDParam(r)

	if err != nil {
		app.badRequestResponse(w, r, err.Error())
		return
	}

	word, err := app.models.Words.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if word.RelatedWords == nil {
		word.RelatedWords = []string{}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"word": word}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) updateWordHandler(w http.ResponseWriter, r *http.Request) {
	// We get the id of the request in the path variable.
	// If we don't get an appropriate id, we send a 404 NOT FOUND.
	id, err := app.readIDParam(r)
	if err != nil {
		app.badRequestResponse(w, r, err.Error())
		return
	}

	// Then we find the word.
	word, err := app.models.Words.Get(id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			slog.Info("Error: ", err.Error())
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Then we extract the new data from the request body
	// and update the word object.
	var input struct {
		Text         string      `json:"text"`
		Difficulty   string      `json:"difficulty"`
		RelatedWords []string    `json:"related_words"`
		UserId       data.UserID `json:"user_id"`
	}

	err = app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err.Error())
		return
	}

	// Partial update
	// Updating the word object with the new values,
	// but only when it has a value.
	if strings.Trim(input.Text, " ") != "" {
		word.TextValue = input.Text
	}

	if strings.Trim(input.Difficulty, " ") != "" {
		word.Difficulty = input.Difficulty
	}

	if input.RelatedWords != nil {
		word.RelatedWords = input.RelatedWords
	}

	if input.UserId > 0 {
		word.UserId = int64(input.UserId)
	}

	// Update the database with the new data
	err = app.models.Words.Update(word)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Notify the client with 200 STATUS OK.
	err = app.writeJSON(w, http.StatusOK, envelope{"word": word}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Delete handler expect the id of the word that is to be deleted.
func (app *application) deleteWordHandler(w http.ResponseWriter, r *http.Request) {
	// First we get the id
	// If the ID is invalid in anyway we send a 404 NOT FOUND.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Delete the word that matches the id.
	err = app.models.Words.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Notify the client that the DELETE operation is complete.
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "word successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listWordsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text         string
		Difficulty   string
		RelatedWords []string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Text = app.readString(qs, "text", "")
	input.Difficulty = app.readString(qs, "difficulty", "")
	input.RelatedWords = app.readCSV(qs, "related_words", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")

	// since our field text_value is only exposed externally as text
	// we need to append _value at the end before sending it to the database query
	if input.Filters.Sort == "text" || input.Filters.Sort == "-text" {
		input.Filters.Sort = input.Filters.Sort + "_value"
	}

	input.Filters.SortSafeList = []string{"id", "text_value", "difficulty", "-id", "-text_value", "-difficulty"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	words, metadata, err := app.models.Words.GetAll(input.Text, input.Difficulty, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"words": words, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
