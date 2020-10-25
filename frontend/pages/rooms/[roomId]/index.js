import {useRouter} from 'next/router'
import styles from "../../../styles/rooms/Rooms.module.scss";
import Head from "next/head";
import {showErrorToastWithError, showSuccessToast, Toast} from "../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import UserRoomListElem from "../userRoomListElem";
import {Button, Spinner} from "react-bootstrap";
import Link from "next/link";

const REFRESH_TIMEOUT = 2000;  // 2s

export default function Room() {
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

  const fetchMusics = () => {
    axiosClient.post('http://localhost:8080/rooms/' + roomId + '/playlists')
      .then(resp => {
        refresh()
        showSuccessToast("Music are currently getting fetched")
      })
      .catch(error => {
        showErrorToastWithError("Failed to find common musics", error)
      })
  }

  useEffect(refresh, [roomId])

  // Do not render anything if no room id exists
  if (!roomId) {
    return null;
  }

  // Force a refresh of the page while we are processing the musics
  if (room.shared_music_library != null && room.shared_music_library.processing_status.success == null) {
    setTimeout(refresh, REFRESH_TIMEOUT)
  }

  let userList = room.users.map(user => {
    return (
      <UserRoomListElem key={user.user_infos.id} user={user} />
    )
  })

  let lock;

  if (room.locked) {
    lock = (
      <p>🔒 Locked</p>
    )

  } else {
    lock = (
      <p>🔓 Unlocked</p>
    )
  }

  let button;

  if (room.shared_music_library == null) {
    button = (
      <Button variant="outline-success" size="lg" className="mt-2 mb-2" onClick={fetchMusics}>
        Find common musics 🎵
      </Button>
    )

  } else if (room.shared_music_library.processing_status.success == null) {
    let current = room.shared_music_library.processing_status.already_processed
    let total = room.shared_music_library.processing_status.total_to_process

    button = (
      <Button variant="warning" size="lg" className="mt-2 mb-2" disabled>
        <Spinner animation="border" className="mr-2"/> Searching common musics ({current}/{total})
      </Button>
    )

  } else if (room.shared_music_library.processing_status.success) {
    button = (
      <Link href={'/rooms/' + roomId + '/playlists'}>
        <Button variant="success" size="lg" className="mt-2 mb-2">
          See common musics ➡️
        </Button>
      </Link>
    )

  } else if (!room.shared_music_library.processing_status.success) {
    button = (
      <Button variant="danger" size="lg" className="mt-2 mb-2" onClick={fetchMusics}>
        ⚰️ An error occurred, try again !
      </Button>
    )
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Shared Spotify</title>
        <link rel="icon" href="/spotify.svg" />
      </Head>

      <main className={styles.main}>
        <h1>Room #{roomId}</h1>

        {lock}

        {button}

        {userList}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}