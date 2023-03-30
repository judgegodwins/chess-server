package websocketconnection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateToken(t *testing.T) {
	t.Run("returns token (happy case)", func(t *testing.T) {
		body := map[string]string{
			"username": "judge",
		}

		request, response := newRequest(t, "/token", body)
		
		testManager.TokenHandler(response, request)
		
		require.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("invalid or no body", func(t *testing.T) {
		body := map[string]string{}

		request, response := newRequest(t, "/token", body)
		
		testManager.TokenHandler(response, request)
		
		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}

func TestAuthMiddlewareAndTokenVerifier(t *testing.T) {
	t.Run("allow valid token entry", func(t *testing.T) {
		token, _, err := testManager.tokenMaker.CreateToken("judge", time.Minute)

		require.NoError(t, err)

		request, response := newRequest(t, "/token/verify", nil)
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		
		testManager.AuthMiddleWare(http.HandlerFunc(testManager.TokenVerifier))(response, request)

		require.Equal(t, http.StatusOK, response.Code)
	})


	t.Run("disallow invalid token entry", func(t *testing.T) {
		token, _, err := testManager.tokenMaker.CreateToken("judge", time.Minute)

		require.NoError(t, err)

		request, response := newRequest(t, "/token/verify", nil)
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token + "hhh"))
		
		testManager.AuthMiddleWare(http.HandlerFunc(testManager.TokenVerifier))(response, request)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})

	t.Run("return unauthorized expired token entry", func(t *testing.T) {
		token, _, err := testManager.tokenMaker.CreateToken("judge", -time.Minute)

		require.NoError(t, err)

		request, response := newRequest(t, "/token/verify", nil)
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
		
		testManager.AuthMiddleWare(http.HandlerFunc(testManager.TokenVerifier))(response, request)

		require.Equal(t, http.StatusUnauthorized, response.Code)
	})
}

func requireBodyMatches[D comparable](t *testing.T, body *bytes.Buffer, value D) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var getData D
	err = json.Unmarshal(data, &getData)

	require.NoError(t, err)
	require.Equal(t, value, getData)
}

func newRequest(t *testing.T, url string, body any) (*http.Request, *httptest.ResponseRecorder) {
	data, err := json.Marshal(body)

	require.NoError(t, err)

	request, err := http.NewRequest(http.MethodPost, "/token", bytes.NewReader(data))
	require.NoError(t, err)

	response := httptest.NewRecorder()

	return request, response
}