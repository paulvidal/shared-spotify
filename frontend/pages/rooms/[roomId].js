import {useRouter} from 'next/router'
import styles from "../../styles/rooms/Rooms.module.scss";
import Head from "next/head";
import {showErrorToastWithError, showSuccessToast, Toast} from "../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import UserRoomListElem from "./userRoomListElem";
import {Button} from "react-bootstrap";

export default function Room() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [room, setRoom] = useState({
    roomId: roomId,
    users: [],
    lock: false
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
    axiosClient.post('http://localhost:8080/rooms/' + roomId + '/musics')
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

  return (
    <div className={styles.container}>
      <Head>
        <title>Shared Spotify</title>
        <link rel="icon" href="/spotify.svg" />
      </Head>

      <main className={styles.main}>
        <h1>Room #{roomId}</h1>

        {lock}

        <Button variant="outline-success" size="lg" className="mt-2 mb-2" onClick={fetchMusics}>
          Find common musics 🎵
        </Button>

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