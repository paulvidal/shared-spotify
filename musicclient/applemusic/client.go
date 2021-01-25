package applemusic

import (
	"context"
	"fmt"
	applemusic "github.com/minchao/go-apple-music"
)

func Run() {
	token := "eyJhbGciOiJFUzI1NiIsImtpZCI6IlY3TUw2UFpQU0oifQ.eyJleHAiOjE2MjY4MzM1ODcsImlhdCI6MTYxMTA1NjU4NywiaXNzIjoiVTRNNVRUTlU1OSJ9.vTTk7JftbMF2GsfljhsYFV9qWSKSNm_CK3J8pctps0Dp0dh4ehnFgLGuH7kbowtYLE3X_xs3L5b_r6jmItYTuQ"
	userToken := "Aqa+mdNHHQZ04Odw4fkCBId82y3hcqUvYjekd/EfxdfC3/ietlrlxLOLeWsKL4DwHDRaoOd+EQuiH/uoT9ekR4mK/o/T206oV1iukv8PYd2DmHfvL6AIa107q6wffMzlOVVzkIogOQ9WSe6uPz/kcYfIEdsUsZR7SG8Mk3dnP8Ncvou3wogFvKIZH1rE3YYs3RQlCHtGmwepyiygJYRjQuoH0KnTWlLESaGxC0jiEQKH2wIT0A=="

	ctx := context.Background()
	tp := applemusic.Transport{Token: token, MusicUserToken: userToken}
	client := applemusic.NewClient(tp.Client())

	st, _, _ := client.Me.GetStorefront(ctx, &applemusic.PageOptions{Offset: 0, Limit: 50})

	for _, s := range st.Data {
		fmt.Println(s)
	}

	s, _, err := client.Catalog.GetSongsByIds(ctx, "fr", []string{"FRZID0900480"}, nil)

	fmt.Println("yo")
	fmt.Println(s)
	fmt.Println(err)

	return

	/**
	Playlists
	*/

	playlists, _, err := client.Me.GetAllLibraryPlaylists(ctx, &applemusic.PageOptions{Offset: 0, Limit: 50})

	if err != nil {
		fmt.Println("Error:")
		fmt.Println(err)
	}

	fmt.Println("Success")
	for _, playlist := range playlists.Data {
		fmt.Println(playlist)
		fmt.Println(playlist.Attributes.CanEdit)

		songs, err := client.Me.GetLibraryPlaylistTracks(ctx, playlist.Id, nil)

		if err != nil {
			fmt.Println("Error:")
			fmt.Println(err)
		}

		fmt.Println("Success got tracks from playlist")

		allSongs := make([]string, 0)

		for _, s := range songs {
			allSongs = append(allSongs, s.Attributes.PlayParams.CatalogId)
		}

		fmt.Println("all songs:", allSongs)

		catalogSongs, _, err := client.Catalog.GetSongsByIds(ctx, "fr", allSongs, &applemusic.Options{})

		if err != nil {
			fmt.Println("Error:")
			fmt.Println(err)
		}

		for _, s := range catalogSongs.Data {
			fmt.Println(s.Attributes.Name, s.Attributes.ArtistName, "ISRC:", s.Attributes.ISRC)
		}
	}

	///**
	//Songs
	//*/
	//
	//songs, _, err := client.Me.GetAllLibrarySongs(ctx, &applemusic.PageOptions{Offset: 0, Limit: 50})
	//
	//if err != nil {
	//	fmt.Println("Error:")
	//	fmt.Println(err)
	//}
	//
	//fmt.Println("Success")
	//
	//allSongs := make([]string, 0)
	//
	//for _, song := range songs.Data {
	//	fmt.Println(song.Attributes.Name, song.Attributes.ArtistName, song.Attributes.AlbumName, song.Id)
	//	allSongs = append(allSongs, song.Attributes.PlayParams.CatalogId)
	//}
	//
	//catalogSongs, _, err := client.Catalog.GetSongsByIds(ctx, "fr", allSongs, nil)
	//
	//if err != nil {
	//	fmt.Println("Error:")
	//	fmt.Println(err)
	//}
	//
	//for _, s := range catalogSongs.Data {
	//	fmt.Println(s.Attributes.Name, s.Attributes.ArtistName, "ISRC:", s.Attributes.ISRC)
	//}
}
