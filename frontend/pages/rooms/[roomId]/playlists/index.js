import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/[roomId]/playlists/Playlists.module.scss"
import {showErrorToastWithError, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import {getUrl} from "../../../../utils/urlUtils";
import CustomHead from "../../../../components/Head";
import Header from "../../../../components/Header";
import {isEmpty} from "lodash";
import PlaylistElem from "../../../../components/playlistElem";
import LoaderScreen from "../../../../components/LoaderScreen";
import {getTotalTrackCount} from "../../../../utils/trackUtils";
import setState from "../../../../utils/stateUtils";
import Footer from "../../../../components/Footer";

export default function Playlists() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlists, setPlaylists] = useState({
    playlists: {},
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
            playlists: playlistsReceived,
            loading: false
          }
        })

      })
      .catch(error => {
        showErrorToastWithError("Failed to get playlists", error, router)
        setState(setPlaylists, {loading: false})
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
  let playlistOrder;

  if (!isEmpty(playlists.playlists)) {

    playlistOrder = Object.keys(playlists.playlists).sort((playlistId1, playlistId2) => {
      let playlist1 = playlists.playlists[playlistId1]
      let playlist2 = playlists.playlists[playlistId2]

      if (playlist1.rank !== playlist2.rank) {
        return playlist1.rank -  playlist2.rank

      } else if (playlist1.rank_for_type !== playlist2.rank_for_type) {
        return playlist1.rank_for_type -  playlist2.rank_for_type
      }

      return getTotalTrackCount(playlists.playlists[playlistId2]) - getTotalTrackCount(playlists.playlists[playlistId1])
    })

    let previousPlaylistType = "";
    let first = true;

    formattedPlaylists = playlistOrder.map((playlistId, index) => {
      let playlist = playlists.playlists[playlistId]

      let title;

      if (playlist.type !== previousPlaylistType) {
        previousPlaylistType = playlist.type

        // Do not show the first playlist title
        if (!first) {
          title = (
            <h5 className="mt-4 mb-2 font-weight-bolder">By {previousPlaylistType}</h5>
          )
        }

        first = false
      }

      return [
        title,
        <PlaylistElem key={playlistId} index={index + 1} roomId={roomId} playlist={playlist}/>
      ]
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

      <Footer/>

      <Toast/>
    </div>
  )
}