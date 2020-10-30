package spotifyclient

import (
	"github.com/zmb3/spotify"
)

type UserInfos struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	ImageUrl string `json:"image"`
}

type User struct {
	Infos  UserInfos       `json:"user_infos"`
	Client *spotify.Client `json:"-"` // we ignore this field
}

func (user *User) GetUserId() string {
	return user.Infos.Name
}

func (user *User) IsEqual(otherUser *User) bool {
	return otherUser.Infos.Id == user.Infos.Id
}