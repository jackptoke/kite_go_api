package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)
	go func() {
		// Create a quit channel which carries os.Signal values
		quit := make(chan os.Signal, 1)
		// Listen for incoming SIGINT and SIGTERM signals
		// and relay them to the quit channel.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		// Block until a signal is received from the quit channel
		s := <-quit

		app.logger.Info("caught signal", "signal", s.String())

		// Create a context with a 30-second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- srv.Shutdown(ctx)
		}

		app.logger.Info("completing background tasks", "addr", srv.Addr)

		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.Info("Starting server", "addr", srv.Addr, "env", app.config.env)
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError

	if err != nil {
		return err
	}

	app.logger.Info("Stopped server", "addr", srv.Addr)
	return nil
}
