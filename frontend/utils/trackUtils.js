import {sum} from "lodash";

function getArtistsFromTrack(track) {
  let artist = 'unknown artist'

  if (track.artists.length > 0) {
    artist = track.artists.map(artist => artist.name).join(", ")
  }

  return artist
}

function getAlbumCoverUrlFromTrack(track) {
  let url = null;

  if (track.album.images.length > 0) {
    url = track.album.images[0].url
  }

  return url
}

function getTotalTrackCount(playlist) {
  return playlist.shared_track_count
}

export {
  getArtistsFromTrack,
  getAlbumCoverUrlFromTrack,
  getTotalTrackCount
}