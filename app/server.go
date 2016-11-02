package app

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/uber-go/zap"
)

// serverErrLogger allows us to use the zap.Logger as our http.Server ErrorLog
type serverErrLogger struct {
	log zap.Logger
}

// Implement Write to log server errors using the zap logger
func (s serverErrLogger) Write(b []byte) (int, error) {
	s.log.Debug(string(b))
	return 0, nil
}

// StartServer starts the web server on the specified port
func (s State) StartServer() error {
	var err error
	server := http.Server{
		ReadTimeout:  s.Config.ReadTimeout,
		WriteTimeout: s.Config.WriteTimeout,
		ErrorLog:     log.New(serverErrLogger{s.Log}, "", 0),
		Handler:      s.Router,
	}

	if len(s.Config.TLSBind) > 0 {
		s.Log.Info("starting https listener", zap.String("bind", s.Config.TLSBind))
		server.Addr = s.Config.TLSBind

		// Redirect http requests to https
		go s.Redirect()
		err = server.ListenAndServeTLS(s.Config.TLSCertFile, s.Config.TLSKeyFile)
	} else {
		s.Log.Info("starting http listener", zap.String("bind", s.Config.Bind))
		server.Addr = s.Config.Bind
		err = server.ListenAndServe()
	}

	return err
}

// Redirect listens on the non-https port, and redirects all requests to https
func (s State) Redirect() {
	var err error

	// Get https port from TLS Bind
	_, httpsPort, err := net.SplitHostPort(s.Config.TLSBind)
	if err != nil {
		s.Log.Fatal("failed to get port from tls bind", zap.Error(err))
	}

	s.Log.Info("starting http -> https redirect listener", zap.String("bind", s.Config.Bind))

	server := http.Server{
		Addr:         s.Config.Bind,
		ReadTimeout:  s.Config.ReadTimeout,
		WriteTimeout: s.Config.WriteTimeout,
		ErrorLog:     log.New(serverErrLogger{s.Log}, "", 0),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Remove port if it exists so we can replace it with https port
			httpHost, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				s.Log.Fatal("failed to get http host from request", zap.Error(err))
			}

			url := fmt.Sprintf("https://%s:%s%s", httpHost, httpsPort, r.RequestURI)
			http.Redirect(w, r, url, http.StatusMovedPermanently)
		}),
	}

	// Start permanent listener
	err = server.ListenAndServe()
	s.Log.Fatal("http redirect listener failed", zap.Error(err))
}