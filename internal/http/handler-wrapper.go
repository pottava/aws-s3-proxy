package http

import (
	"compress/gzip"
	"compress/zlib"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pottava/aws-s3-proxy/internal/config"
)

// WrapHandler wraps every handlers
func WrapHandler(handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := config.Config

		// If there is a health check path defined, and if this path matches it,
		// then return 200 OK and return.
		if len(c.HealthCheckPath) > 0 && r.URL.Path == c.HealthCheckPath {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Omiting healtcheck
		if c.DebugOutput {
			log.Printf("[debug]: URI requested %s", r.RequestURI)
		}
		// WhiteListIPs
		if len(c.WhiteListIPRanges) > 0 {
			clientIP := getIP(r)
			if c.DebugOutput {
				log.Printf("[debug]: Client IP: %s", clientIP)
			}
			found := false
			for _, whiteListIPRange := range c.WhiteListIPRanges {
				ip := net.ParseIP(clientIP)
				found = whiteListIPRange.Contains(ip)
				if found {
					break
				}
			}
			if !found {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
		}
		// CORS
		if (len(c.CorsAllowOrigin) > 0) &&
			(len(c.CorsAllowMethods) > 0) &&
			(len(c.CorsAllowHeaders) > 0) &&
			(c.CorsMaxAge > 0) {
			w.Header().Set("Access-Control-Allow-Origin", c.CorsAllowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", c.CorsAllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", c.CorsAllowHeaders)
			w.Header().Set("Access-Control-Max-Age", strconv.FormatInt(c.CorsMaxAge, 10))
		}
		// BasicAuth
		if (len(c.BasicAuthUser) > 0) && (len(c.BasicAuthPass) > 0) &&
			!auth(r, c.BasicAuthUser, c.BasicAuthPass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="REALM"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		// Auth with JWT
		if len(c.JwtSecretKey) > 0 && !isValidJwt(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="REALM"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		proc := time.Now()
		addr := r.RemoteAddr
		if ip, found := header(r, "X-Forwarded-For"); found {
			addr = ip
		}
		// Content-Encoding
		ioWriter := w.(io.Writer)
		if encodings, found := header(r, "Accept-Encoding"); found && c.ContentEncoding {
			for _, encoding := range splitCsvLine(encodings) {
				if encoding == "gzip" {
					w.Header().Set("Content-Encoding", "gzip")
					g := gzip.NewWriter(w)
					defer g.Close()
					ioWriter = g
					break
				}
				if encoding == "deflate" {
					w.Header().Set("Content-Encoding", "deflate")
					z := zlib.NewWriter(w)
					defer z.Close()
					ioWriter = z
					break
				}
			}
		}
		// Handle HTTP requests
		writer := &custom{Writer: ioWriter, ResponseWriter: w, status: http.StatusOK}
		handler(writer, r)

		// AccessLog
		if c.AccessLog {
			log.Printf("[%s] %.3f %d %s %s",
				addr, time.Since(proc).Seconds(),
				writer.status, r.Method, r.URL)
		}
	})
}

// getIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	retIP := r.RemoteAddr
	if forwarded != "" {
		retIP = forwarded
	}
	//Lets remove port - if present
	slices := strings.Split(retIP, ":")
	if len(slices) > 1 {
		retIP = strings.Join(slices[:len(slices)-1], ":")
	}
	return retIP
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
	splitted := strings.SplitN(data, ",", -1)
	parsed := make([]string, len(splitted))
	for i, val := range splitted {
		parsed[i] = strings.TrimSpace(val)
	}
	return parsed
}

func isValidJwt(r *http.Request) bool {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")
	if len(splitToken) != 2 {
		// Error: Bearer token not in proper format
		return false
	}
	reqToken = strings.TrimSpace(splitToken[1])
	token, err := jwt.Parse(reqToken, func(t *jwt.Token) (interface{}, error) {
		secretKey := config.Config.JwtSecretKey
		return []byte(secretKey), nil
	})
	return err == nil && token.Valid
}
