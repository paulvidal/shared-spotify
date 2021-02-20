package applemusic

import (
	"context"
	applemusic "github.com/minchao/go-apple-music"
	"github.com/shared-spotify/datadog"
	"github.com/shared-spotify/logger"
	"github.com/shared-spotify/musicclient/clientcommon"
)

func GetStorefront(user *clientcommon.User) (*string, error) {
	client := user.AppleMusicClient

	storefronts, _, err := client.Me.GetStorefront(
		context.Background(),
		&applemusic.PageOptions{Offset: 0, Limit: maxPage})

	clientcommon.SendRequestMetric(datadog.AppleRequest, datadog.RequestTypeUserInfo, true, err)

	if err != nil {
		logger.WithUser(user.GetUserId()).Error("Failed to fetch storefront for apple user ", err)
		return nil, err
	}

	logger.WithUser(user.GetUserId()).Infof("Found %d storefronts for apple user", len(storefronts.Data))

	// We always take the first one, users should generally only have 1
	storefront := storefronts.Data[0].Id

	return &storefront, nil
}
