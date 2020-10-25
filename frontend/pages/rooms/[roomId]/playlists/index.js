import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/Rooms.module.scss";
import Head from "next/head";
import {showErrorToastWithError, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import PlaylistElem from "./playlistElem";
import ReactAudioPlayer from "react-audio-player";
import {Button, Tooltip, OverlayTrigger} from "react-bootstrap";

export default function Playlist() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlists, setPlaylists] = useState({
    tracks_in_common: [],
    song_playing: ''
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId) {
      return null;
    }

    axiosClient.get('http://localhost:8080/rooms/' + roomId + '/playlists')
      .then(resp => {
        setPlaylists(prevState => {
          return {
            ...prevState,
            ...resp.data,
          }
        })
      })
      .catch(error => {
        showErrorToastWithError("Failed to get playlists", error)
      })
  }

  useEffect(refresh, [roomId])

  // Do not render anything if no room id exists
  if (!roomId) {
    return null;
  }

  const updateSongCallback = (song) => {
    setPlaylists(prevState => {
      return {
        ...prevState,
        song_playing: song
      }
    })
  }

  let music = (
    <h3 className="mt-5 text-center">No track in commons found... 😞</h3>
  );

  if (playlists.tracks_in_common) {
    music = playlists.tracks_in_common.sort((track1, track2) => {
      return track1.artists[0].name.localeCompare(track2.artists[0].name)
    }).map(track => {
      return (
        <PlaylistElem
          key={track.id}
          track={track}
          songPlaying={playlists.song_playing}
          updateSongCallback={updateSongCallback}/>
      )
    })
  }

  let player = (
    <ReactAudioPlayer
      src={playlists.song_playing}
      autoPlay
    />
  )

  const playlistName = `Shared spotify - Room #${roomId}"`

  let info;

  if (playlists.tracks_in_common) {
    info = [
      (
        <p className="font-weight-bold">
          {playlists.tracks_in_common.length} songs in common
        </p>
      ),
      (
        <OverlayTrigger
          key="top"
          placement="top"
          overlay={
            <Tooltip id={`tooltip-top`}>
              Created playlist will be called "{playlistName}"
            </Tooltip>
          }
        >
          <Button variant="outline-success" className="mb-3">
            Add to my playlists
          </Button>
        </OverlayTrigger>
      )
    ]
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Shared Spotify</title>
        <link rel="icon" href="/spotify.svg" />
      </Head>

      <main className={styles.main}>
        <h1>Playlists</h1>
        <p>Room #{roomId}</p>
        {info}
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