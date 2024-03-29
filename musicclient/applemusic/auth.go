package applemusic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/httputils"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/mongoclient"
	"github.com/shared-spotify/musicclient/clientcommon"
	"github.com/shared-spotify/utils"
	"net/http"
	"net/url"
	"time"
)

type AppleLogin struct {
	UserId            string `json:"user_id"`
	UserEmail         string `json:"user_email"`
	UserName          string `json:"user_name"`
	MusickitToken     string `json:"musickit_token"`
	MusicKitUserToken string `json:"musickit_user_token"`
}

type AppleUser struct {
	UserId    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

// This is used to immediately insert apple user when this one is collected in the frontend
func UserHandler(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Received new apple user")
	datadog.Increment(1, datadog.UserLoginStarted, datadog.Provider.Tag(datadog.AppleMusicProvider))

	var user AppleUser
	err := httputils.DeserialiseBody(r, &user)

	if err != nil {
		logger.Logger.Error("Failed to decode json body for for apple user info")
		http.Error(w, "Bad user info sent for apple user", http.StatusBadRequest)
		return
	}

	if user.UserName == "" || user.UserId == "" {
		logger.Logger.Error("Bad apple user info ", user)
		http.Error(w, "Bad apple user info", http.StatusBadRequest)
		return
	}

	// Add the user in mongo if it did not exist
	mongoUser := &clientcommon.User{UserInfos: &clientcommon.UserInfos{Id: user.UserId, Name: user.UserName,
		Email: user.UserEmail, JoinDate: time.Now()}, LoginType: clientcommon.AppleMusicLoginType}
	err = mongoclient.InsertUsers([]*clientcommon.User{mongoUser})

	if err != nil {
		logger.Logger.Error("Failed to insert apple user in mongo ", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	httputils.SendOk(w)
}

// the user will eventually be redirected back to your redirect URL
func CallbackHandler(w http.ResponseWriter, r *http.Request) {
	logger.Logger.Info("Apple authentication request received")

	queryValues := r.URL.Query()

	userId := queryValues.Get("user_id")
	userEmail := queryValues.Get("user_email")
	userName := queryValues.Get("user_name")
	musickitToken := queryValues.Get("musickit_token")
	musicKitUserToken := queryValues.Get("musickit_user_token")

	logger.Logger.Info(userId, userEmail, userName, musickitToken, musicKitUserToken)

	appleLogin := AppleLogin{
		userId,
		userEmail,
		userName,
		musickitToken,
		musicKitUserToken}

	cookie, err := EncryptToken(&appleLogin)
	if err != nil {
		logger.Logger.Error("Failed to set token", err)
		http.Error(w, "Failed to set token", http.StatusBadRequest)
		return
	}

	_, err = CreateUserFromToken(&appleLogin, cookie.Value)

	if err != nil {
		logger.Logger.Error("Failed to authenticate user with apple music ", err)
		http.Error(w, "Failed to authenticate user with apple music", http.StatusBadRequest)
		return
	}

	// Add the token as an encrypted cookie
	http.SetCookie(w, cookie)

	// Add the login type cookie name
	loginTypeCookie, err := clientcommon.GetLoginTypeCookie(clientcommon.AppleMusicLoginType)
	if err != nil {
		logger.Logger.Error("Failed to set loginType", err)
		http.Error(w, "Failed to set loginType", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, loginTypeCookie)

	redirectUri := queryValues.Get("redirect_uri")
	redirect := clientcommon.FrontendUrl

	if redirectUri != "" {
		redirect = clientcommon.FrontendUrl + redirectUri
	}

	logger.Logger.Info("Redirecting to ", redirect)
	datadog.Increment(1, datadog.UserLoginSuccess, datadog.Provider.Tag(datadog.AppleMusicProvider))

	http.Redirect(w, r, redirect, http.StatusFound)
}

func CreateUserFromToken(appleLogin *AppleLogin, tokenStr string) (*clientcommon.User, error) {
	userInfos := clientcommon.UserInfos{Id: appleLogin.UserId, Name: appleLogin.UserName, ImageUrl: "",
		Email: appleLogin.UserEmail, JoinDate: time.Now()}

	// Create the apple music client
	tp := applemusic.Transport{Token: appleLogin.MusickitToken, MusicUserToken: appleLogin.MusicKitUserToken}
	client := &http.Client{
		Transport: &tp,
		Timeout:   time.Second * ClientTimeout,
	}
	appleMusicClient := applemusic.NewClient(client)

	// make a dummy request to make sure token is valid
	_, _, err := appleMusicClient.Me.GetStorefront(context.Background(), nil)

	clientcommon.SendRequestMetric(datadog.AppleMusicProvider, datadog.RequestTypeUserInfo, true, nil)

	if err != nil {
		logger.Logger.Warning("Invalid apple music user token ", err)
		return nil, err
	}

	user := &clientcommon.User{
		UserInfos:        &userInfos,
		AppleMusicClient: appleMusicClient,
		LoginType:        clientcommon.AppleMusicLoginType,
		Token:            tokenStr,
	}

	// Get the name for the user
	users, err := mongoclient.GetUsers([]string{user.GetId()})

	if err != nil {
		logger.Logger.Error("Failed to get user in mongo ", err)
		return nil, errors.New("Failed to create user")
	}

	if mongoUser, ok := users[appleLogin.UserId]; ok {
		// Get the name from mongo if user already exists
		user.Name = mongoUser.Name

	} else {
		// Make sure we add something for the name if not present
		if user.Name == "" {
			user.Name = appleLogin.UserEmail
		}

		// Add the user in mongo if did not exist
		err := mongoclient.InsertUsers([]*clientcommon.User{user})

		if err != nil {
			logger.Logger.Error("Failed to insert apple user in mongo ", err)
			return nil, errors.New("Failed to create user")
		}
	}

	return user, nil
}

func EncryptToken(appleLogin *AppleLogin) (*http.Cookie, error) {
	jsonToken, err := json.Marshal(*appleLogin)

	if err != nil {
		logger.Logger.Error("Failed to serialise json apple login")
		return nil, err
	}

	encryptedToken, err := utils.Encrypt(jsonToken, clientcommon.TokenEncryptionKey)

	if err != nil {
		logger.Logger.Error("Failed to encrypt apple token ", err)
		return nil, err
	}

	base64EncryptedToken := base64.StdEncoding.EncodeToString(encryptedToken)

	expiration := time.Now().Add(365 * 24 * time.Hour)

	urlParsed, err := url.Parse(clientcommon.BackendUrl)

	if err != nil {
		logger.Logger.Error("Failed to parse urls ", err)
		return nil, err
	}

	secure := true
	sameSite := http.SameSiteNoneMode

	// for localhost development
	if urlParsed.Scheme == "http" {
		secure = false
		sameSite = http.SameSiteDefaultMode
	}

	cookie := http.Cookie{
		Name:    clientcommon.TokenCookieName,
		Value:   base64EncryptedToken,
		Expires: expiration,
		// we send the cookie cross domain, so we need all this
		Domain:   urlParsed.Host,
		Path:     "/",
		Secure:   secure,
		SameSite: sameSite,
	}

	return &cookie, nil
}

func DecryptToken(tokenStr string) (*AppleLogin, error) {
	var appleLogin AppleLogin

	base64JsonToken, err := base64.StdEncoding.DecodeString(tokenStr)

	if err != nil {
		logger.Logger.Error("Failed to decode base64 apple token ", err)
		return nil, err
	}

	decryptedToken, err := utils.Decrypt(base64JsonToken, clientcommon.TokenEncryptionKey)

	if err != nil {
		logger.Logger.Error("Failed to decrypt apple token ", err)
		return nil, err
	}

	err = json.Unmarshal(decryptedToken, &appleLogin)

	if err != nil {
		logger.Logger.Error("Failed to deserialise json apple token ", err)
		return nil, err
	}

	return &appleLogin, nil
}
