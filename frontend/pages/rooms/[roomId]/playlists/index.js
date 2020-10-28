import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/[roomId]/playlists/Playlists.module.scss"
import {showErrorToastWithError, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import {getUrl} from "../../../../utils/urlUtils";
import CustomHead from "../../../../components/Head";
import Header from "../../../../components/Header";
import {isEmpty} from "lodash";
import PlaylistListElem from "../../../../components/playlistListElem";
import {min, max} from 'lodash'
import {Form} from 'react-bootstrap'
import {getTrackBackground, Range} from "react-range";


export default function Playlists() {
  const router = useRouter()
  const { roomId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlists, setPlaylists] = useState({
    playlists: {},
    minFriends: 0,
    minFriendsLimit: 0,
    maxFriendsLimit: 0
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

          let minFriends = min(Object.values(playlistsReceived.playlists).map(playlist => playlist.shared_count))
          let maxFriends = max(Object.values(playlistsReceived.playlists).map(playlist => playlist.shared_count))

          return {
            ...prevState,
            ...playlistsReceived,
            minFriends: maxFriends,
            minFriendsLimit: minFriends,
            maxFriendsLimit: maxFriends
          }
        })

      })
      .catch(error => {
        showErrorToastWithError("Failed to get playlists", error)
      })
  }

  useEffect(refresh, [roomId])

  let formattedPlaylists;

  if (!isEmpty(playlists.playlists)) {

    formattedPlaylists = Object.keys(playlists.playlists).sort((playlistId1, playlistId2) => {
      return playlists.playlists[playlistId1].name.localeCompare(playlists.playlists[playlistId2].name)

    }).filter(playlistId => {
      return playlists.playlists[playlistId].shared_count >= playlists.minFriends

    }).map((playlistId, index) => {
      let playlist = playlists.playlists[playlistId]

      return (
        <PlaylistListElem key={playlistId} index={index + 1} roomId={roomId} playlist={playlist}/>
      )
    })
  }

  let slider;
  let sliderHelp;

  // Ugly slider but it does the job
  if (roomId && playlists.minFriendsLimit !== playlists.maxFriendsLimit) {
    slider = (
      <Range
        step={1}
        min={playlists.minFriendsLimit}
        max={playlists.maxFriendsLimit}
        values={[playlists.minFriends]}
        onChange={(values) => {
          setPlaylists(prevState => {
            return {
              ...prevState,
              minFriends: values[0]
            }
          })
        }}
        renderTrack={({ props, children }) => (
          <div
            onMouseDown={props.onMouseDown}
            onTouchStart={props.onTouchStart}
            style={{
              ...props.style,
              height: '36px',
              display: 'flex',
              width: '50%'
            }}
          >
            <div
              ref={props.ref}
              style={{
                height: '5px',
                width: '100%',
                borderRadius: '4px',
                background: getTrackBackground({
                  values: [playlists.minFriends],
                  colors: ['#28a745', '#cccccc'],
                  min: playlists.minFriendsLimit,
                  max: playlists.maxFriendsLimit
                }),
                alignSelf: 'center'
              }}
            >
              {children}
            </div>
          </div>
        )}
        renderThumb={({ props, isDragged }) => (
          <div
            {...props}
            style={{
              ...props.style,
              height: '42px',
              width: '42px',
              borderRadius: '4px',
              backgroundColor: '#ffffff',
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              boxShadow: '0px 2px 6px #AAA'
            }}
          >
            <div
              style={{
                position: 'absolute',
                top: '-35px',
                color: '#fff',
                fontWeight: 'bold',
                fontSize: '14px',
                fontFamily: 'Arial,Helvetica Neue,Helvetica,sans-serif',
                padding: '4px',
                paddingLeft: '7px',
                paddingRight: '7px',
                borderRadius: '4px',
                backgroundColor: '#28a745'
              }}
            >
              {playlists.minFriends.toFixed(0)}
            </div>
            <div
              style={{
                height: '16px',
                width: '5px',
                backgroundColor: isDragged ? '#28a745' : '#CCC'
              }}
            />
          </div>
        )}
      />
    )

    sliderHelp = (
      <p className="mb-5 mt-2 text-center">Select the minimum number of person that <br/>
      must have the song in their library for it to be chosen</p>
    )
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className="mb-5">Playlists</h1>
        {slider}
        {sliderHelp}

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