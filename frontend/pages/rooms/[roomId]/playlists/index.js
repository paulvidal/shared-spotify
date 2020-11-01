import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/[roomId]/playlists/Playlists.module.scss"
import {showErrorToastWithError, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import {getUrl} from "../../../../utils/urlUtils";
import CustomHead from "../../../../components/Head";
import Header from "../../../../components/Header";
import {isEmpty, sum} from "lodash";
import PlaylistElem from "../../../../components/playlistElem";
import LoaderScreen from "../../../../components/LoaderScreen";
import {getTotalTrackCount} from "../../../../utils/trackUtils";
import setState from "../../../../utils/stateUtils";

export default function Playlists() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlists, setPlaylists] = useState({
    playlist_types: {},
    loading: true
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId) {
      return null;
    }

    axiosClient.get(getUrl('/rooms/' + roomId + '/playlists'))
      .then(resp => {
        let playlistsReceived = resp.data

        setPlaylists(prevState => {
          return {
            ...prevState,
            ...playlistsReceived,
            loading: false
          }
        })

      })
      .catch(error => {
        setState(setPlaylists, {loading: false})
        showErrorToastWithError("Failed to get playlists", error)
      })
  }

  useEffect(refresh, [roomId])

  // Use a loader screen if nothing is ready
  if (playlists.loading) {
    return (
      <LoaderScreen/>
    )
  }

  let formattedPlaylists;

  if (!isEmpty(playlists.playlist_types)) {

    formattedPlaylists = Object.keys(playlists.playlist_types).sort((playlistId1, playlistId2) => {
      let playlist1 = playlists.playlist_types[playlistId1]
      let playlist2 = playlists.playlist_types[playlistId2]

      if (playlist1.rank !== playlist2.rank) {
        return playlist1.rank -  playlist2.rank
      }

      return getTotalTrackCount(playlists.playlist_types[playlistId2]) - getTotalTrackCount(playlists.playlist_types[playlistId1])

    }).map((playlistId, index) => {
      let playlist = playlists.playlist_types[playlistId]

      return (
        <PlaylistElem key={playlistId} index={index + 1} roomId={roomId} playlist={playlist}/>
      )
    })
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className="mb-5 text-center">Generated Playlists</h1>
        {formattedPlaylists}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>
    </div>
  )
}