/*
Copyright © 2020 FairOS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cookie

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

const (
	cookieName           = "fairOS-dfs"
	cookieSessionId      = "session-id"
	cookieLoginTime      = "login-time"
	cookieExpirationTime = 24 * time.Hour
	cookieLogoutTime     = 1 * time.Hour
)

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

// GetUniqueSessionId generates a sessionId for each logged-in user
func GetUniqueSessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// SetSession sets the cookie in the http.Response
func SetSession(sessionId string, response http.ResponseWriter, cookieDomain string) error {
	logoutTime := time.Now().Add(cookieLogoutTime)
	logoutTimeStr := logoutTime.Format(time.RFC3339)
	value := map[string]string{
		cookieSessionId: sessionId,
		cookieLoginTime: logoutTimeStr,
	}
	encoded, err := cookieHandler.Encode(cookieName, value)
	if err != nil {
		return err
	}

	expire := time.Now().Add(cookieExpirationTime)
	var cookie *http.Cookie
	if cookieDomain == "localhost" {
		cookie = &http.Cookie{
			Name:    cookieName,
			Value:   encoded,
			Path:    "/",
			Expires: expire,
			//SameSite: http.SameSiteNoneMode, // Can't have SameSite || Secure on localhost or cookie will be blocked
			//Secure:   true, // Can't have SameSite || Secure on localhost or cookie will be blocked
			//HttpOnly: true, // Can't have this on localhost or cookie will be inaccessibe via document.cookie
			MaxAge: 0, // to make sure that the browser does not persist it in disk
		}
	} else if cookieDomain == "" {
		cookie = &http.Cookie{
			Name:     cookieName,
			Value:    encoded,
			Path:     "/",
			Expires:  expire,
			HttpOnly: true,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			MaxAge:   0, // to make sure that the browser does not persist it in disk
		}
	} else {
		cookie = &http.Cookie{
			Name:     cookieName,
			Value:    encoded,
			Path:     "/",
			Expires:  expire,
			HttpOnly: true,
			Domain:   cookieDomain,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
			MaxAge:   0, // to make sure that the browser does not persist it in disk
		}
	}

	http.SetCookie(response, cookie)
	return nil
}

// GetSessionIdFromCookie extracts sessionId from http.Request
func GetSessionIdFromCookie(request *http.Request) (sessionId string, err error) {
	cookie, err := request.Cookie(cookieName)
	if err != nil {
		return "", err
	}
	cookieValue := make(map[string]string)
	err = cookieHandler.Decode(cookieName, cookie.Value, &cookieValue)
	if err != nil {
		return "", err
	}
	sessionId = cookieValue[cookieSessionId]
	return sessionId, nil
}

// GetSessionIdFromRawCookie extracts sessionId from a raw cookie string
func GetSessionIdFromRawCookie(cookieStr string) (sessionId string, err error) {
	cookieValue := make(map[string]string)
	err = cookieHandler.Decode(cookieName, cookieStr, &cookieValue)
	if err != nil {
		return "", err
	}
	sessionId = cookieValue[cookieSessionId]
	return sessionId, nil
}

// GetSessionIdAndLoginTimeFromCookie extracts sessionId and logged-in time from http.Request
func GetSessionIdAndLoginTimeFromCookie(request *http.Request) (sessionId, loginTime string, err error) {
	cookie, err := request.Cookie(cookieName)
	if err != nil {
		return "", "", err
	}
	cookieValue := make(map[string]string)
	err = cookieHandler.Decode(cookieName, cookie.Value, &cookieValue)
	if err != nil {
		return "", "", err
	}
	sessionId = cookieValue[cookieSessionId]
	loginTime = cookieValue[cookieLoginTime]
	return sessionId, loginTime, nil
}

// ClearSession clears the cookie from http.Request
func ClearSession(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Expires:  time.Now().Add(-time.Duration(1) * time.Second), // set the expiry to 1 second
	}
	http.SetCookie(response, cookie)
}
