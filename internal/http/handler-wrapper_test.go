package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/pottava/aws-s3-proxy/internal/config"
	"github.com/stretchr/testify/assert"
)

const sample = "http://example.com/foo"

func TestWithoutAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, sample, nil)
	assert.False(t, auth(req, "user", "pass"))
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestWithoutBasic(t *testing.T) {
	username := "user"
	password := "pass"

	req := httptest.NewRequest(http.MethodGet, sample, nil)
	req.Header.Set("Authorization", basicAuth(username, password))

	assert.False(t, auth(req, username, password))
}

func TestAuthMatch(t *testing.T) {
	username := "user"
	password := "pass"

	req := httptest.NewRequest(http.MethodGet, sample, nil)
	req.Header.Set("Authorization", "Basic "+basicAuth(username, password))

	assert.True(t, auth(req, username, password))
}

func TestWithValidJWT(t *testing.T) {
	username := "user"
	password := "pass"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"password": password,
	})
	tokenString, _ := token.SignedString([]byte("secret"))
	c := config.Config
	c.JwtSecretKey = "secret"
	req := httptest.NewRequest(http.MethodGet, sample, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))

	assert.True(t, isValidJwt(req))
}

func TestWithoutValidJWT(t *testing.T) {
	username := "user"
	password := "pass"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"password": password,
	})
	tokenString, _ := token.SignedString([]byte("secret"))
	c := config.Config
	c.JwtSecretKey = "foo"
	req := httptest.NewRequest(http.MethodGet, sample, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenString))

	assert.False(t, isValidJwt(req))
}

func TestHeaderWithValue(t *testing.T) {
	expected := "test"

	req := httptest.NewRequest(http.MethodGet, sample, nil)
	req.Header.Set("Test", expected)

	actual, found := header(req, "Test")

	assert.True(t, found)
	assert.Equal(t, expected, actual)
}

func TestHeaderWithoutValue(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, sample, nil)
	_, found := header(req, "Test")
	assert.False(t, found)
}

func TestSplitCsvLine(t *testing.T) {
	expected := 3

	lines := splitCsvLine("1,2,3")

	assert.Equal(t, expected, len(lines))
}

func TestTrimedSplitCsvLine(t *testing.T) {
	expected := 3

	lines := splitCsvLine("1 , 2 ,3 ")

	assert.Equal(t, expected, len(lines))
	assert.Equal(t, "1", lines[0])
	assert.Equal(t, "2", lines[1])
	assert.Equal(t, "3", lines[2])
}
