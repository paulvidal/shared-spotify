package clientcommon

import (
	"github.com/minchao/go-apple-music"
	"github.com/zmb3/spotify"
)

type UserInfos struct {
	Id       string `json:"id" bson:"_id"`
	Name     string `json:"name"`
	ImageUrl string `json:"image"`
}

type User struct {
	*UserInfos       `bson:"inline"`
	SpotifyClient    *spotify.Client    `json:"-"` // we ignore this field
	AppleMusicClient *applemusic.Client `json:"-"` // we ignore this field
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
