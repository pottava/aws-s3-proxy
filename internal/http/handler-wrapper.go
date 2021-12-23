package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

// WrapHandler wraps every handlers
func WrapHandler(handler echo.HandlerFunc) echo.HandlerFunc {
	return echo.HandlerFunc(func(e echo.Context) error {
		c := config.Config
		req := e.Request()
		res := e.Response()

		// If there is a health check path defined, and if this path matches it,
		// then return 200 OK and return.
		if len(c.HealthCheckPath) > 0 && req.URL.Path == c.HealthCheckPath {
			res.WriteHeader(http.StatusOK)
			return nil
		}

		proc := time.Now()
		addr := req.RemoteAddr

		// BasicAuth
		if (len(c.BasicAuthUser) > 0) && (len(c.BasicAuthPass) > 0) &&
			!auth(req, c.BasicAuthUser, c.BasicAuthPass) {
			res.Header().Set("WWW-Authenticate", `Basic realm="REALM"`)
			http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

			c.Logger.Infof("[%s] %.3f %d %s %s",
				addr, time.Since(proc).Seconds(),
				http.StatusUnauthorized, req.Method, req.URL)

			return nil
		}

		// Facility Header if set
		if len(c.Facility) > 0 {
			res.Header().Add("Facility", c.Facility)
		}

		return handler(e)
	})
}

func auth(r *http.Request, authUser, authPass string) bool {
	if username, password, ok := r.BasicAuth(); ok {
		return username == authUser && password == authPass
	}

	return false
}

func header(r *http.Request, key string) (string, bool) {
	if r.Header == nil {
		return "", false
	}

	if candidate := r.Header[key]; len(candidate) > 0 {
		return candidate[0], true
	}

	return "", false
}

func splitCsvLine(data string) []string {
	splitted := strings.Split(data, ",")
	parsed := make([]string, len(splitted))

	for i, val := range splitted {
		parsed[i] = strings.TrimSpace(val)
	}

	return parsed
}
