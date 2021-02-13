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
import Footer from "../../../components/Footer";

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
    refreshing: false,
    errorRefresh: false,
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
          refreshing: false,
          errorRefresh: false,
          loading: false
        })
      })
      .catch(error => {
        let hasAlreadySeenRefreshedError = room.errorRefresh
        setState(setRoom, {
          refreshing: false,
          errorRefresh: true,
          loading: false
        })
        if (!hasAlreadySeenRefreshedError) {
          showErrorToastWithError("Failed to get room info", error, router)
        }
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
        showErrorToastWithError("Failed to find common musics", error, router)
      })
  }

  useEffect(refresh, [roomId])

  // Use a loader screen if nothing is ready
  if (room.loading) {
    return (
      <LoaderScreen />
    )
  }

  // Handle refresh of the page
  let timeout = null

  if (room.users && room.users.length === 0) {
    timeout = null;  // do not refresh when the room is not found

  } else if (!room.shared_music_library) {
    timeout = GENERAL_REFRESH_TIMEOUT

  } else if (room.shared_music_library.processing_status.success == null) {
    // Force a refresh of the page while we are processing the musics more often to get the progress
    timeout = REFRESH_TIMEOUT_PLAYLIST_CREATION
  }

  if (timeout && !room.refreshing) {
    setState(setRoom, {refreshing: true})
    setTimeout(refresh, timeout)
  }

  let userList = room.users.map(user => {
    return (
      <UserRoomListElem key={user.id} user={user} />
    )
  })

  let lock;

  if (room.locked) {
    lock = (
      <p className="text-center ml-2 mr-2">
        🔒 Locked<br/>
        (room is not accepting new members)
      </p>
    )

  } else {
    lock = (
      <p className="text-center">
        🔓 Open
      </p>
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
                Only {room.owner.name}, the person that created the room, can trigger this action
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
          See songs in common ➡️
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

  let shareButton;

  if (!room.locked) {
    shareButton = (
      <CopyToClipboard text={process.env.NEXT_PUBLIC_URL + '/rooms/' + roomId + '/share'}
                       onCopy={() => showSuccessToast("Invite link copied to clipboard")}>
        <Button variant="outline-warning" className="mt-2 mb-2">Add friends to room 🔗</Button>
      </CopyToClipboard>
    )
  }

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

      <Footer/>

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