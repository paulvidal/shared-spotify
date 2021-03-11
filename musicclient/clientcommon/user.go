package clientcommon

import (
	"github.com/minchao/go-apple-music"
	"github.com/patrickmn/go-cache"
	"github.com/shared-spotify/logger"
	"github.com/zmb3/spotify"
	"time"
)

// Define a user cache so that we don't recreate clients all the time
var userCache = cache.New(10 * time.Minute, 20 * time.Minute)

type UserInfos struct {
	Id       string `json:"id" bson:"_id"`
	Name     string `json:"name"`
	ImageUrl string `json:"image"`
	Email    string `json:"-" bson:"email"`
}

type User struct {
	*UserInfos       `bson:"inline"`
	SpotifyClient    *spotify.Client    `json:"-"` // we ignore this field
	AppleMusicClient *applemusic.Client `json:"-"` // we ignore this field
	LoginType        string             `json:"-" bson:"login_type"`
	Token            string             `json:"-" bson:"token"`
}

func (user *User) GetId() string {
	return user.Id
}

func (user *User) GetUserId() string {
	return user.Name
}

func (user *User) IsEqual(otherUser *User) bool {
	return otherUser.Id == user.Id
}

/**
  Determine music provider
*/

func (user *User) IsSpotify() bool {
	return user.LoginType == SpotifyLoginType || user.SpotifyClient != nil
}

func (user *User) IsAppleMusic() bool {
	return user.LoginType == AppleMusicLoginType || user.AppleMusicClient != nil
}

/**
  User Cache
 */

func AddUserToCache(token string, user *User) {
	logger.WithUser(user.GetUserId()).Debug("User was not found in cache, created it")
	userCache.SetDefault(token, user)
}

func GetUserFromCache(token string) (*User, bool) {
	if entry, ok := userCache.Get(token); ok {
		user := entry.(*User)
		logger.WithUser(user.GetUserId()).Debug("Loaded user from cache")
		return user, true
	}

	return nil, false
}