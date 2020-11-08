import {useRouter} from 'next/router'
import styles from "../../../styles/rooms/[roomId]/Room.module.scss";
import {showErrorToastWithError, showSuccessToast, Toast} from "../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import UserRoomListElem from "../../../components/userRoomListElem";
import {OverlayTrigger, Tooltip, Button, Spinner} from "react-bootstrap";
import Link from "next/link";
import {getUrl} from "../../../utils/urlUtils";
import {CopyToClipboard} from "react-copy-to-clipboard";
import CustomHead from "../../../components/Head";
import Header from "../../../components/Header";
import LoaderScreen from "../../../components/LoaderScreen";
import CustomModal from "../../../components/CustomModal";
import setState from "../../../utils/stateUtils";

const GENERAL_REFRESH_TIMEOUT = 6000;  // 6s
const REFRESH_TIMEOUT_PLAYLIST_CREATION = 2000;  // 2s

const MIN_USERS_TO_SHARE = 2;

export default function Room() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [room, setRoom] = useState({
    roomId: roomId,
    name: '',
    is_owner: false,
    owner: {},
    users: [],
    locked: false,
    shared_music_library: null,
    awaiting_new_refresh: true,
    stop_refresh: false,
    loading: true,
    showConfirmationModal: false
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId) {
      return null;
    }

    axiosClient.get(getUrl('/rooms/' + roomId))
      .then(resp => {
        setState(setRoom, {
          ...resp.data,
          loading: false
        })
      })
      .catch(error => {
        setState(setRoom, {
          stop_refresh: true,
          loading: false
        })
        showErrorToastWithError("Failed to get room info", error)
      })
  }

  const showModal = () => {
    setState(setRoom, {showConfirmationModal: true})
  }

  const hideModal = () => {
    setState(setRoom, {showConfirmationModal: false})
  }

  const fetchMusics = () => {
    hideModal()

    axiosClient.post(getUrl('/rooms/' + roomId + '/playlists'))
      .then(resp => {
        refresh()
      })
      .catch(error => {
        showErrorToastWithError("Failed to find common musics", error)
      })
  }

  useEffect(refresh, [roomId])

  // Use a loader screen if nothing is ready
  if (room.loading) {
    return (
      <LoaderScreen />
    )
  }

  // we refresh the page on a daily basis waiting for users to join
  if (room.awaiting_new_refresh) {

    if (room.shared_music_library != null && room.shared_music_library.processing_status.success == null) {
      // Force a refresh of the page while we are processing the musics more often to get the progress
      setTimeout(() => {
        setState(setRoom, {awaiting_new_refresh: true})
        refresh()
      }, REFRESH_TIMEOUT_PLAYLIST_CREATION)

    } else if (!room.stop_refresh && !room.locked) {
      setTimeout(() => {
        setState(setRoom, {awaiting_new_refresh: true})
        refresh()
      }, GENERAL_REFRESH_TIMEOUT)
    }

    setState(setRoom, {awaiting_new_refresh: false})
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
      <p>🔓 Open</p>
    )
  }

  let button;

  if (room.shared_music_library == null) {

    if (room.users.length >= MIN_USERS_TO_SHARE) {

      if (room.is_owner) {
        button = (
          <Button variant="success" size="lg" className="mt-2 mb-2" onClick={showModal}>
            Find common music 🎵
          </Button>
        )

      } else {
        button = (
          <OverlayTrigger
            key="overlay"
            placement="top"
            overlay={
              <Tooltip id="overlay-tooltip">
                Only {room.owner.user_infos.name}, the person that created the room, can trigger this action
              </Tooltip>
            }
          >
            <div>
              <Button variant="success" size="lg" className="mt-2 mb-2 disabled">
                Find common music 🎵
              </Button>
            </div>
          </OverlayTrigger>
        )
      }
    }

  } else if (room.shared_music_library.processing_status.success == null) {
    let current = room.shared_music_library.processing_status.already_processed
    let total = room.shared_music_library.processing_status.total_to_process

    button = (
      <Button variant="warning" size="lg" className="mt-2 mb-2" disabled>
        <Spinner animation="border" className="mr-2"/> Searching common musics ({Math.floor(current/total*100)}%)
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
      <Button variant="danger" size="lg" className="mt-2 mb-2" onClick={showModal} disabled={!room.is_owner}>
        ⚰️ An error occurred, try again !
      </Button>
    )
  }

  let shareButton = (
    <CopyToClipboard text={process.env.NEXT_PUBLIC_URL + '/rooms/' + roomId + '/share'}
                     onCopy={() => showSuccessToast("Shareable link copied to clipboard")}>
      <Button variant="outline-warning" className="mt-2 mb-2">Share room 🔗</Button>
    </CopyToClipboard>
  )

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1>{room.name}</h1>

        {lock}

        {shareButton}

        {button}

        {userList}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>

      <CustomModal
        show={room.showConfirmationModal}
        body={"Finding the common musics will close the room, so no more people will be able to join. " +
        "Are you sure you want to do this now?"}
        secondaryActionName={"Cancel"}
        secondaryAction={hideModal}
        onHideAction={hideModal}
        primaryActionName={"Find musics"}
        primaryAction={fetchMusics}
      />
    </div>
  )
}