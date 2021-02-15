package auth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func createTestAuth(t *testing.T, url string) *Auth {
	t.Helper()

	a, err := New("pages.gitlab-example.com",
		"something-very-secret",
		"id",
		"secret",
		"http://pages.gitlab-example.com/auth",
		url,
		"scope")

	require.NoError(t, err)

	return a
}

type domainMock struct {
	projectID       uint64
	notFoundContent string
}

func (dm *domainMock) GetProjectID(r *http.Request) uint64 {
	return dm.projectID
}

func (dm *domainMock) ServeNotFoundAuthFailed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(dm.notFoundContent))
}

// Gorilla's sessions use request context to save session
// Which makes session sharable between test code and actually manipulating session
// Which leads to negative side effects: we can't test encryption, and cookie params
// like max-age and secure are not being properly set
// To avoid that we use fake request, and set only session cookie without copying context
func setSessionValues(t *testing.T, r *http.Request, store sessions.Store, values map[interface{}]interface{}) {
	t.Helper()

	tmpRequest, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	result := httptest.NewRecorder()

	session, _ := store.Get(tmpRequest, "gitlab-pages")
	session.Values = values
	session.Save(tmpRequest, result)

	for _, cookie := range result.Result().Cookies() {
		r.AddCookie(cookie)
	}
}

func TestTryAuthenticate(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something/else")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	require.Equal(t, false, auth.TryAuthenticate(result, r, source.NewMockSource()))
}

func TestTryAuthenticateWithError(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?error=access_denied")
	require.NoError(t, err)

	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, 401, result.Code)
}

func TestTryAuthenticateWithCodeButInvalidState(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=invalid")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["state"] = "state"
	session.Save(r, result)

	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, 401, result.Code)
}

func TestTryAuthenticateRemoveTokenFromRedirect(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state&token=secret")
	require.NoError(t, err)

	require.Equal(t, reqURL.Query().Get("token"), "secret", "token is present before redirecting")
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["state"] = "state"
	session.Values["proxy_auth_domain"] = "https://domain.com"
	session.Save(r, result)

	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, http.StatusFound, result.Code)

	redirect, err := url.Parse(result.Header().Get("Location"))
	require.NoError(t, err)

	require.Empty(t, redirect.Query().Get("token"), "token is gone after redirecting")
}

func testTryAuthenticateWithCodeAndState(t *testing.T, https bool) {
	t.Helper()

	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			require.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "{\"access_token\":\"abc\"}")
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	domain := apiServer.URL
	if https {
		domain = strings.Replace(apiServer.URL, "http://", "https://", -1)
	}

	code, err := auth.EncryptAndSignCode(domain, "1")
	require.NoError(t, err)

	r, err := http.NewRequest("GET", "/auth?code="+code+"&state=state", nil)
	require.NoError(t, err)
	if https {
		r.URL.Scheme = request.SchemeHTTPS
	} else {
		r.URL.Scheme = request.SchemeHTTP
	}

	r.Host = strings.TrimPrefix(apiServer.URL, "http://")

	setSessionValues(t, r, auth.store, map[interface{}]interface{}{
		"uri":   "https://pages.gitlab-example.com/project/",
		"state": "state",
	})

	result := httptest.NewRecorder()
	require.Equal(t, true, auth.TryAuthenticate(result, r, source.NewMockSource()))
	require.Equal(t, http.StatusFound, result.Code)
	require.Equal(t, "https://pages.gitlab-example.com/project/", result.Header().Get("Location"))
	require.Equal(t, 600, result.Result().Cookies()[0].MaxAge)
	require.Equal(t, https, result.Result().Cookies()[0].Secure)
}

func TestTryAuthenticateWithCodeAndStateOverHTTP(t *testing.T) {
	testTryAuthenticateWithCodeAndState(t, false)
}

func TestTryAuthenticateWithCodeAndStateOverHTTPS(t *testing.T) {
	testTryAuthenticateWithCodeAndState(t, true)
}

func TestCheckAuthenticationWhenAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	session.Save(r, result)
	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.False(t, contentServed)

	// notFoundContent wasn't served so the default response from CheckAuthentication should be 200
	require.Equal(t, 200, result.Code)
}

func TestCheckAuthenticationWhenNoAccess(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	w := httptest.NewRecorder()

	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	session.Save(r, w)

	contentServed := auth.CheckAuthentication(w, r, &domainMock{projectID: 1000, notFoundContent: "Generic 404"})
	require.True(t, contentServed)
	res := w.Result()
	defer res.Body.Close()

	require.Equal(t, 404, res.StatusCode)

	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, string(body), "Generic 404")
}

func TestCheckAuthenticationWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/projects/1000/pages_access":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	err = session.Save(r, result)
	require.NoError(t, err)

	contentServed := auth.CheckAuthentication(result, r, &domainMock{projectID: 1000})
	require.True(t, contentServed)
	require.Equal(t, 302, result.Code)
}

func TestCheckAuthenticationWithoutProject(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	reqURL.Scheme = request.SchemeHTTPS
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.False(t, contentServed)
	require.Equal(t, 200, result.Code)
}

func TestCheckAuthenticationWithoutProjectWhenInvalidToken(t *testing.T) {
	apiServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/user":
			require.Equal(t, "Bearer abc", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "{\"error\":\"invalid_token\"}")
		default:
			t.Logf("Unexpected r.URL.RawPath: %q", r.URL.Path)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	apiServer.Start()
	defer apiServer.Close()

	auth := createTestAuth(t, apiServer.URL)

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/auth?code=1&state=state")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	session.Save(r, result)

	contentServed := auth.CheckAuthenticationWithoutProject(result, r, &domainMock{projectID: 0})
	require.True(t, contentServed)
	require.Equal(t, 302, result.Code)
}

func TestGenerateKeys(t *testing.T) {
	keys, err := generateKeys("something-very-secret", 3)
	require.NoError(t, err)
	require.Len(t, keys, 3)

	require.NotEqual(t, fmt.Sprint(keys[0]), fmt.Sprint(keys[1]))
	require.NotEqual(t, fmt.Sprint(keys[0]), fmt.Sprint(keys[2]))
	require.NotEqual(t, fmt.Sprint(keys[1]), fmt.Sprint(keys[2]))

	require.Equal(t, len(keys[0]), 32)
	require.Equal(t, len(keys[1]), 32)
	require.Equal(t, len(keys[2]), 32)
}

func TestGetTokenIfExistsWhenTokenExists(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Values["access_token"] = "abc"
	session.Save(r, result)

	token, err := auth.GetTokenIfExists(result, r)
	require.NoError(t, err)
	require.Equal(t, "abc", token)
}

func TestGetTokenIfExistsWhenTokenDoesNotExist(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("http://pages.gitlab-example.com/test")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL, Host: "pages.gitlab-example.com", RequestURI: "/test"}

	session, err := auth.store.Get(r, "gitlab-pages")
	require.NoError(t, err)

	session.Save(r, result)

	token, err := auth.GetTokenIfExists(result, r)
	require.Equal(t, "", token)
	require.Equal(t, nil, err)
}

func TestCheckResponseForInvalidTokenWhenInvalidToken(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("http://pages.gitlab-example.com/test")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL, Host: "pages.gitlab-example.com", RequestURI: "/test"}

	resp := &http.Response{StatusCode: http.StatusUnauthorized, Body: ioutil.NopCloser(bytes.NewReader([]byte("{\"error\":\"invalid_token\"}")))}

	require.Equal(t, true, auth.CheckResponseForInvalidToken(result, r, resp))
	require.Equal(t, http.StatusFound, result.Result().StatusCode)
	require.Equal(t, "http://pages.gitlab-example.com/test", result.Header().Get("Location"))
}

func TestCheckResponseForInvalidTokenWhenNotInvalidToken(t *testing.T) {
	auth := createTestAuth(t, "")

	result := httptest.NewRecorder()
	reqURL, err := url.Parse("/something")
	require.NoError(t, err)
	r := &http.Request{URL: reqURL}

	resp := &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewReader([]byte("ok")))}

	require.Equal(t, false, auth.CheckResponseForInvalidToken(result, r, resp))
}
