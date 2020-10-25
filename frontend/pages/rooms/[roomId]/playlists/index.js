import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/Rooms.module.scss";
import Head from "next/head";
import {showErrorToastWithError, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";

export default function Playlist() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [room, setRoom] = useState({
    roomId: roomId,
    users: [],
    lock: false,
    shared_music_library: null
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId) {
      return null;
    }

    axiosClient.get('http://localhost:8080/rooms/' + roomId)
      .then(resp => setRoom(resp.data))
      .catch(error => {
        showErrorToastWithError("Failed to get room info", error)
      })
  }

  useEffect(refresh, [roomId])

  // Do not render anything if no room id exists
  if (!roomId) {
    return null;
  }

  let music = null;

  if (room.shared_music_library && room.shared_music_library.common_playlists) {
    music = room.shared_music_library.common_playlists.tracks_in_common.map(track => {
      let artist = 'unknown'

      if (track.artists.length > 0) {
        artist = track.artists.map(artist => artist.name).join(", ")
      }

      return (
        <li key={track.id}>{track.name} - {artist}
          <audio controls>
            <source src={track.preview_url} type="audio/mpeg"/>
          </audio>
        </li>
      )
    })
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

        <ol className="mt-2 col-10 col-md-5">
          {music}
        </ol>
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}