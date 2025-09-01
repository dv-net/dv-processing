package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dv-net/dv-processing/internal/util"
	"github.com/dv-net/mx/logger"
	"github.com/urfave/cli/v2"

	mxsignal "github.com/dv-net/mx/util/signal"
)

/*

	Webhooks HTTP Server

	Use the following server to create a simple HTTP server that listens for POST requests on the /webhook endpoint.
	When a request is received, the server will log the request body and respond with a 202 Accepted status code
	Useful for testing webhooks service.

*/

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), mxsignal.Shutdown()...)
	defer cancel()

	// init default logger
	l := logger.NewExtended(
		logger.WithConsoleColored(true),
		logger.WithLogFormat(logger.LoggerFormatConsole),
	)

	app := &cli.App{
		Name:    "Webhooks testing server",
		Version: "develop",
		Suggest: true,
		Commands: []*cli.Command{
			startCMD(l),
		},
	}

	// run cli runner
	if err := app.RunContext(ctx, os.Args); err != nil {
		l.Fatalf("failed to run cli runner: %s", err)
	}
}

func startCMD(l logger.ExtendedLogger) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "start the server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Usage: "address to listen",
				Value: ":8085",
			},
			&cli.StringFlag{
				Name:  "endpoint",
				Usage: "endpoint to handle requests",
				Value: "/webhook",
			},
			&cli.StringFlag{
				Name:    "secret",
				Aliases: []string{"s"},
				Usage:   "Client secret key",
			},
			&cli.BoolFlag{
				Name:    "check-signature",
				Aliases: []string{"cs"},
				Usage:   "allows to check the signature of the request",
			},
		},
		Action: func(c *cli.Context) error {
			// start http server with post webhook handler
			http.HandleFunc(c.String("endpoint"), func(w http.ResponseWriter, r *http.Request) {
				fn := func() error {
					// check method
					if r.Method != http.MethodPost {
						return fmt.Errorf("invalid method: %s", r.Method)
					}

					// read body
					body, err := io.ReadAll(r.Body)
					if err != nil {
						return fmt.Errorf("read body: %w", err)
					}
					defer func() {
						if err := r.Body.Close(); err != nil {
							l.Errorf("close body: %s", err)
						}
					}()

					// check signature
					if c.Bool("check-signature") {
						if r.Header.Get("X-Sign") != util.SHA256Signature(body, c.String("secret")) {
							return fmt.Errorf("invalid signature")
						}
					}

					l.Infof("webhook received: %v", string(body))

					return nil
				}

				if err := fn(); err != nil {
					l.Errorf("webhook error: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte(err.Error()))
					if err != nil {
						l.Errorf("write response: %s", err)
					}
					return
				}

				w.WriteHeader(http.StatusAccepted)
				_, err := w.Write([]byte("ok"))
				if err != nil {
					l.Errorf("write response: %s", err)
				}
			})

			srv := &http.Server{
				Addr:         c.String("addr"),
				WriteTimeout: 1 * time.Second,
				ReadTimeout:  1 * time.Second,
			}

			errChan := make(chan error, 1)
			go func() {
				errChan <- srv.ListenAndServe()
			}()

			l.Infow("webhooks server started", "addr", c.String("addr"), "endpoint", c.String("endpoint"), "callback_url", fmt.Sprintf("http://localhost%s%s", c.String("addr"), c.String("endpoint")))

			select {
			case <-c.Done():
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				l.Info("shutting down the server")

				if err := srv.Shutdown(ctx); err != nil {
					return fmt.Errorf("failed to shutdown server: %w", err)
				}

				return nil
			case err := <-errChan:
				return err
			}
		},
	}
}
