package main

import (
	"expvar"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	// Custom error handler for 404 Not Found responses.
	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	// Custom error handler for 405 Method Not Allowed responses.
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/api/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/api/v1/words", app.requirePermission("words:read", app.listWordsHandler))
	router.HandlerFunc(http.MethodPost, "/api/v1/words", app.requirePermission("words:write", app.createWordHandler))
	router.HandlerFunc(http.MethodGet, "/api/v1/words/:id", app.requirePermission("words:read", app.getWordHandler))
	router.HandlerFunc(http.MethodPatch, "/api/v1/words/:id", app.requirePermission("words:write", app.updateWordHandler))
	router.HandlerFunc(http.MethodDelete, "/api/v1/words/:id", app.requirePermission("words:write", app.deleteWordHandler))

	router.HandlerFunc(http.MethodPost, "/api/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/api/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPost, "/api/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	router.Handler(http.MethodGet, "/debug/vars", expvar.Handler())
	// this whole api now has CORS enabled.
	//return app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router))))
	return app.metrics(app.recoverPanic(app.rateLimit(app.authenticate(router))))
}
