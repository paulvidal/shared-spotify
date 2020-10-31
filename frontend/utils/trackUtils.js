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

function getTotalTrackCount(trackType) {
  return sum(
    Object.keys(trackType.tracks_per_shared_count)
      .map(sharedCount => trackType.tracks_per_shared_count[sharedCount].length)
  )
}

export {
  getArtistsFromTrack,
  getAlbumCoverUrlFromTrack,
  getTotalTrackCount
}