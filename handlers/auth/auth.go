package auth

import (
	"encoding/base64"
	"net/http"

	"github.com/RideShare-Server/log"
	"github.com/RideShare-Server/utils"
	"github.com/jinzhu/gorm"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/juju/errors"
	"github.com/justinas/nosurf"
	"github.com/volatiletech/authboss"
	// To enable the auth and lock modules, they need to be imported
	_ "github.com/volatiletech/authboss/auth"
	_ "github.com/volatiletech/authboss/lock"
	aboauth2 "github.com/volatiletech/authboss/oauth2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// RootURL :
	RootURL = "AUTH_ROOT_URL"
	// LoginOkPath :
	LoginOkPath = "AUTH_LOGIN_OK_PATH"
	// LogoutOkPath :
	LogoutOkPath = "AUTH_LOGOUT_OK_PATH"

	// CookieStoreKey :
	CookieStoreKey = "AUTH_COOKIE_STORE_KEY"
	// SessionStoreKey :
	SessionStoreKey = "AUTH_SESSION_STORE_KEY"

	// GoogleOAuthClientID :
	GoogleOAuthClientID = "GOOGLE_OAUTH_CLIENT_ID"
	// GoogleOAuthClientSecret :
	GoogleOAuthClientSecret = "GOOGLE_OAUTH_CLIENT_SECRET"
)

var (
	ab = authboss.New()
)

// buildCookieStore sets up the cookieStore
func buildCookieStore() {
	// Get the CookieStoreKey from the environment variables, and decode it
	encodedCookieKey := utils.GetEnvVariable(CookieStoreKey, "")
	if encodedCookieKey == "" {
		log.Error(log.AuthTopic, errors.Errorf("Empty Cookie Store Key"))
	}
	cookieStoreKey, err := base64.StdEncoding.DecodeString(encodedCookieKey)
	if err != nil {
		e := errors.Annotate(err, "Cookie Store Key Error")
		log.Error(log.AuthTopic, e)
	}
	cookieStore = securecookie.New(cookieStoreKey, nil)
}

// buildSessionCookieStore sets up the sessionStore
func buildSessionCookieStore() {
	// Get the SessionStoreKey from the environment variables, and decode it
	encodedSessionKey := utils.GetEnvVariable(SessionStoreKey, "")
	if encodedSessionKey == "" {
		log.Error(log.AuthTopic, errors.Errorf("Empty Session Store Key"))
	}
	sessionStoreKey, err := base64.StdEncoding.DecodeString(encodedSessionKey)
	if err != nil {
		e := errors.Annotate(err, "Session Store Key Error")
		log.Error(log.AuthTopic, e)
	}
	sessionStore = sessions.NewCookieStore(sessionStoreKey)
}

// getOAuth2Providers returns a map of providers to use with AuthBoss
func getOAuth2Providers() map[string]authboss.OAuth2Provider {
	return map[string]authboss.OAuth2Provider{
		"google": authboss.OAuth2Provider{
			OAuth2Config: &oauth2.Config{
				ClientID:     utils.GetEnvVariable(GoogleOAuthClientID, ""),
				ClientSecret: utils.GetEnvVariable(GoogleOAuthClientSecret, ""),
				Scopes:       []string{`profile`, `email`},
				Endpoint:     google.Endpoint,
			},
			Callback: aboauth2.Google,
		},
	}
}

// SetupAuth sets up the auth package
func SetupAuth(database *gorm.DB) *authboss.Authboss {
	// Build the Cookie Store and Session Store
	buildCookieStore()
	buildSessionCookieStore()

	googleAuthDataStore := NewGoogleAuthStore(database)

	ab.Storer = googleAuthDataStore
	ab.OAuth2Storer = googleAuthDataStore
	ab.MountPath = "/auth"
	// TODO: Fix url fetch to accommodate different domains.
	ab.RootURL = utils.GetEnvVariable(RootURL, "http://localhost:8888")
	ab.AuthLoginOKPath = utils.GetEnvVariable(LoginOkPath, "http://localhost:8080/login_success")
	ab.AuthLogoutOKPath = utils.GetEnvVariable(LogoutOkPath, "http://localhost:8080/logout_success/ok")
	ab.OAuth2Providers = getOAuth2Providers()

	ab.XSRFName = "csrf_token"
	ab.XSRFMaker = func(_ http.ResponseWriter, r *http.Request) string {
		return nosurf.Token(r)
	}

	ab.CookieStoreMaker = NewCookieStore
	ab.SessionStoreMaker = NewSessionStore

	if err := ab.Init(); err != nil {
		log.Error(log.AuthTopic, err)
	}
	return ab
}
