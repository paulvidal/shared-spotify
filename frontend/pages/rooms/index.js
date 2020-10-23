import styles from "../../styles/rooms/Rooms.module.scss";
import Head from "next/head";
import {useState} from "react";
import useDeepCompareEffect from 'use-deep-compare-effect'
import axios from "axios";
import RoomListElem from "./roomListElem";
import { Button } from 'react-bootstrap';

import {Toast, showSuccessToast, showErrorToast} from "../../components/toast";

export default function Rooms() {
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [rooms, setRooms] = useState([]);

  const refresh = () => {
    axiosClient.get('http://localhost:8080/rooms')
      .then(resp => setRooms(Object.values(resp.data.rooms)))
      .catch(error => {
        showErrorToast("Failed to get rooms info")
      })
  }

  const createRoom = () => {
    axiosClient.post('http://localhost:8080/rooms')
      .then(resp => {
        showSuccessToast("Room successfully created !")
        refresh()
      })
      .catch(error => {
        showErrorToast("Room failed to create ! Please try again")
      })
  }

  useDeepCompareEffect(refresh, [rooms])

  let roomsList;

  if (rooms.length === 0) {
    roomsList = (
      <p className="mt-4">No rooms at the moment...</p>
    )

  } else {
    roomsList = rooms.map(room => {
      return (
        <RoomListElem key={room.id} room={room}/>
      )
    });
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Shared Spotify</title>
        <link rel="icon" href="/spotify.svg" />
      </Head>

      <main className={styles.main}>
        <h1>Rooms</h1>

        {roomsList}

        <Button variant="outline-success" size="lg" className="mt-4" onClick={createRoom}>
          Create a new room
        </Button>
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}