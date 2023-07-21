package api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var oauthConfig *oauth2.Config
var randomCode string
var tokenReady = make(chan interface{}, 1)
var shutdownServer = make(chan interface{}, 1)
var apiToken string
var srv *http.Server

func init() {
	randomCode = getRandomCode()
}

func getRandomCode() string {
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to generate random code")
	}
	return string(b)
}

func InitOAuthConfig(apiURL, clientID, clientSecret string) {
	oauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:9090/auth/callback",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/authenticate", apiURL),
			TokenURL: fmt.Sprintf("%s/authenticate/token", apiURL),
		},
	}
}

func startAuthServer() {
	router := http.NewServeMux()
	router.HandleFunc("/auth/callback", callBackHandler)
	router.HandleFunc("/login", loginHandler)

	srv = &http.Server{
		Addr:    ":9090",
		Handler: router,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.WithError(err).Fatal("Auth server failed")
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL(randomCode)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callBackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	state := r.FormValue("state")

	if state != randomCode {
		logrus.Fatal("State is incorrect")
	}

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to exchange token")
	}

	apiToken = token.AccessToken
	tokenReady <- nil
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func oauthFlow(apiURL, clientID, clientSecret string) string {
	InitOAuthConfig(apiURL, clientID, clientSecret)
	go startAuthServer()

	go func() {
		<-shutdownServer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logrus.WithError(err).Fatal("Failed to shutdown auth server")
		}
	}()

	defer func() {
		shutdownServer <- nil
	}()

	err := open("http://localhost:9090/login")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open browser")
	}

	<-tokenReady

	return apiToken
}

func getOAuthCredentials() (string, string, error) {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return "", "", errors.New("CLIENT_ID or CLIENT_SECRET not set")
	}
	return clientID, clientSecret, nil
}
