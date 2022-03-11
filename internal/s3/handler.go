package s3

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/packethost/aws-s3-proxy/internal/config"
)

// Handler wraps every the controllers
func Handler(handler echo.HandlerFunc) echo.HandlerFunc {
	return echo.HandlerFunc(func(e echo.Context) error {
		h := config.Cfg.HTTPOpts
		req := e.Request()
		res := e.Response()

		// If there is a health check path defined, and if this path matches it,
		// then return 200 OK and return.
		if h.HealthCheckPath != "" && req.URL.Path == h.HealthCheckPath {
			res.WriteHeader(http.StatusOK)
			return nil
		}

		// Facility Header if set
		if h.Facility != "" {
			res.Header().Add("Facility", h.Facility)
		}

		return handler(e)
	})
}
