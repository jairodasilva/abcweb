package middleware

import (
	"bytes"
	"fmt"
	"net/http"

	chimiddleware "github.com/pressly/chi/middleware"
)

// Recover middleware recovers panics that occur and gracefully logs their error
func (m Middleware) Recover(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); r != nil {
				recover()
				buf := &bytes.Buffer{}
				requestID := chimiddleware.GetReqID(r.Context())

				if requestID != "" {
					fmt.Fprintf(buf, "[%s] ", requestID)
				}

				fmt.Fprintf(buf, `"%s `, r.Method)

				if r.TLS == nil {
					if _, err := buf.WriteString(`http`); err != nil {
						panic(err)
					}
				} else {
					if _, err := buf.WriteString(`https`); err != nil {
						panic(err)
					}
				}

				fmt.Fprintf(buf, "://%s%s %s\" from %s -- panic:\n%+v", r.Host, r.RequestURI, r.Proto, r.RemoteAddr, err)

				m.Log.Error(buf.String())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
