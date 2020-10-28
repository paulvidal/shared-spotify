import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/Rooms.module.scss";
import {showErrorToastWithError, showSuccessToast, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import PlaylistElem from "../../../../components/playlistElem";
import ReactAudioPlayer from "react-audio-player";
import {Button, OverlayTrigger, Spinner, Tooltip} from "react-bootstrap";
import {getArtistsFromTrack} from "../../../../utils/trackUtils";
import {isEmpty} from "lodash"
import {getUrl} from "../../../../utils/urlUtils";
import CustomHead from "../../../../components/Head";
import Header from "../../../../components/Header";

export default function Playlist() {
  const router = useRouter()
  const { roomId, playlistId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlist, setPlaylist] = useState({
    name: '',
    tracks: [],
    song_playing: '',
    creating_playlist: false,
    created_playlist: {}
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId || !playlistId) {
      return null;
    }

    axiosClient.get(getUrl('/rooms/' + roomId + '/playlists/' + playlistId))
      .then(resp => {
        setPlaylist(prevState => {
          return {
            ...prevState,
            ...resp.data,
          }
        })
      })
      .catch(error => {
        showErrorToastWithError("Failed to get playlist " + playlistId, error)
      })
  }

  useEffect(refresh, [roomId, playlistId])

  const addPlaylist = () => {
    let confirmation = confirm("You are creating a playlist on your account, do you wish to continue?")

    if (!confirmation) {
      return
    }

    setPlaylist(prevState => {
      return {
        ...prevState,
        creating_playlist: true,
      }
    })

    axiosClient.post(getUrl('/rooms/' + roomId + '/playlists/' + playlistId + '/add'))
      .then(resp => {
        const playlistName = resp.data.name

        setPlaylist(prevState => {
          return {
            ...prevState,
            creating_playlist: false,
            new_playlist: resp.data
          }
        })
        showSuccessToast(`Successfully created in spotify playlist "${playlistName}"`)
      })
      .catch(error => {
        showErrorToastWithError("Failed to create playlist in spotify", error)
      })
      .finally(() => {
        setPlaylist(prevState => {
          return {
            ...prevState,
            creating_playlist: false,
          }
        })
      })
  }

  const updateSongCallback = (song) => {
    setPlaylist(prevState => {
      return {
        ...prevState,
        song_playing: song
      }
    })
  }

  let music = (
    <h3 className="mt-5 text-center">No tracks in common found... 😞</h3>
  );

  if (playlist.tracks) {
    music = playlist.tracks.sort((track1, track2) => {
      return getArtistsFromTrack(track1).localeCompare(getArtistsFromTrack(track2))
    }).map(track => {
      return (
        <PlaylistElem
          key={track.id}
          track={track}
          songPlaying={playlist.song_playing}
          updateSongCallback={updateSongCallback}/>
      )
    })
  }

  let player = (
    <ReactAudioPlayer
      src={playlist.song_playing}
      autoPlay
    />
  )

  let info;
  let addButton;

  if (playlist.tracks) {
    info = (
      <p className="font-weight-bold">
        {playlist.tracks.length} songs in common 🎉
      </p>
    )

    if (playlist.creating_playlist) {
      addButton = (
        <Button variant="warning" size="lg" className="mb-3" disabled>
          <Spinner animation="border" className="mr-2"/> Creating playlist
        </Button>
      )

    } else if (!isEmpty(playlist.created_playlist)) {
      let url = "#"

      if (playlist.created_playlist.spotify_url) {
        url = playlist.created_playlist.spotify_url
      }

      addButton = (
        (
          <Button variant="success" size="lg" className="mb-3" target="_blank" href={url}>
            Go to my new playlist ➡️
          </Button>
        )
      )

    } else {
      addButton = (
        (
          <OverlayTrigger
            key="top"
            placement="top"
            overlay={
              <Tooltip id={`tooltip-top`}>
                Playlist will be created in spotify and added to your playlists
              </Tooltip>
            }
          >
            <Button variant="outline-success" size="lg" className="mb-3" onClick={addPlaylist}>
              Add to my playlists
            </Button>
          </OverlayTrigger>
        )
      )
    }
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h3 className="text-center mt-3 mb-3">{playlist.name}</h3>
        <p>Room #{roomId}</p>
        {info}
        {addButton}
        {music}
        {player}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}