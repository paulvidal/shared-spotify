import {useRouter} from 'next/router'
import styles from "../../../../styles/rooms/[roomId]/playlists/Playlist.module.scss";
import {showErrorToastWithError, showSuccessToast, Toast} from "../../../../components/toast";
import axios from "axios";
import {useEffect, useState} from "react";
import PlaylistListElem from "../../../../components/playlistListElem";
import ReactAudioPlayer from "react-audio-player";
import {Button, OverlayTrigger, Spinner, Tooltip} from "react-bootstrap";
import {getArtistsFromTrack} from "../../../../utils/trackUtils";
import {isEmpty, max, min, sum} from "lodash"
import {getUrl} from "../../../../utils/urlUtils";
import CustomHead from "../../../../components/Head";
import Header from "../../../../components/Header";
import {getTrackBackground, Range} from "react-range";
import LoaderScreen from "../../../../components/LoaderScreen";
import CustomModal from "../../../../components/CustomModal";
import setState from "../../../../utils/stateUtils";

const TIMEOUT_BEFORE_BUTTON_AVAILABLE = 2000  // 2s
const IDEAL_DEFAULT_COUNT = 40

function findBestDefaultSharedCount(playlists) {
  let playlistSharedCount = Object.keys(playlists)
    .sort()
    .reverse()

  let currentTrackCount = 0;
  let sharedCount = null;

  for (let i = 0; i < playlistSharedCount.length; i++) {
    sharedCount = playlistSharedCount[i]
    let tracks = playlists[sharedCount]

    currentTrackCount += tracks.length

    if (currentTrackCount >= IDEAL_DEFAULT_COUNT) {
      break
    }
  }

  return sharedCount
}

export default function Playlist() {
  const router = useRouter()
  const { roomId, playlistId } = router.query

  const axiosClient = axios.create({
    withCredentials: true
  })

  const [playlist, setPlaylist] = useState({
    type: '',
    tracks_per_shared_count: {},
    song_playing: '',
    creating_playlist: false,
    new_playlist: {},
    minSharedCount: 0,
    minSharedCountLimit: 0,
    maxSharedCountLimit: 0,
    loading: true,
    showConfirmationModal: false,
    user_ids_per_shared_tracks: {},
    users: {}
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId || !playlistId) {
      return null;
    }

    axiosClient.get(getUrl('/rooms/' + roomId + '/playlists/' + playlistId))
      .then(resp => {
        let playlistReceived = resp.data.tracks_per_shared_count

        let sharedCounts = Object.keys(playlistReceived).map(i => {
          return parseInt(i)
        })

        let minSharedCountLimit = min(sharedCounts)
        let maxSharedCountLimit = max(sharedCounts)

        let bestDefaultSharedCount = findBestDefaultSharedCount(playlistReceived)
        if (!bestDefaultSharedCount) {
          bestDefaultSharedCount = maxSharedCountLimit
        }

        setState(setPlaylist, {
          ...resp.data,
          minSharedCount: parseInt(bestDefaultSharedCount),
          minSharedCountLimit: minSharedCountLimit,
          maxSharedCountLimit: maxSharedCountLimit,
          loading: false
        })
      })
      .catch(error => {
        setState(setPlaylist, {loading: false})
        showErrorToastWithError("Failed to get playlist " + playlistId, error)
      })
  }

  useEffect(refresh, [roomId, playlistId])

  // Use a loader screen if nothing is ready
  if (playlist.loading) {
    return (
      <LoaderScreen/>
    )
  }

  const showModal = () => {
    setState(setPlaylist, {showConfirmationModal: true})
  }

  const hideModal = () => {
    setState(setPlaylist, {showConfirmationModal: false})
  }

  const addPlaylist = () => {
    setState(setPlaylist, {
      showConfirmationModal: false,
      creating_playlist: true
    })

    const minSharedCount =  playlist.minSharedCount

    axiosClient.post(getUrl('/rooms/' + roomId + '/playlists/' + playlistId + '/add'), {
      min_shared_count: minSharedCount
    }).then(resp => {
      setState(setPlaylist, {
        new_playlist: {
          [minSharedCount]: resp.data
        }
      })

      setTimeout(() => {
        setState(setPlaylist, {creating_playlist: false})
      }, TIMEOUT_BEFORE_BUTTON_AVAILABLE)
    })
    .catch(error => {
      setState(setPlaylist, {creating_playlist: false})
      showErrorToastWithError("Failed to create playlist in spotify", error)
    })
  }

  const updateSongCallback = (song) => {
    setState(setPlaylist, {song_playing: song})
  }

  let tracksPerSharedCount = Object.keys(playlist.tracks_per_shared_count)
    .filter(sharedCount => parseInt(sharedCount) >= playlist.minSharedCount)
    .sort()
    .reverse()

  let trackTotalCount = sum(
    tracksPerSharedCount
      .map(sharedCount => playlist.tracks_per_shared_count[sharedCount].length)
  )

  let info;

  let music = (
    <p className="mt-5 text-center font-weight-bold">No tracks in common found... 😞</p>
  );

  if (trackTotalCount !== 0) {

    info = [
      <p key="count" className="font-weight-bold text-center mb-0">
        {trackTotalCount} songs in common 🎉
      </p>,
      <p key="info" className="font-weight-normal">
        (shared between at least {playlist.minSharedCount} friends)
      </p>
    ]

    music = tracksPerSharedCount
      .map((sharedCount, index) => {
        let tracks = playlist.tracks_per_shared_count[sharedCount]

        let divider;

        if (index !== tracksPerSharedCount.length - 1) {
          divider = (
            <div className={styles.group_divider + " mt-5 col-5 col-md-3"}/>
          )
        }

        return (
          <div key={sharedCount} className={styles.common_songs_group}>
            <h5 className="mt-3 mb-3">Songs shared by {sharedCount} friends</h5>
            {tracks.sort((track1, track2) => {
                return getArtistsFromTrack(track1).localeCompare(getArtistsFromTrack(track2))
              })
              .map(track => {
                let trackISRC = track.external_ids["isrc"]
                let userIds = playlist.user_ids_per_shared_tracks[trackISRC]
                let users = userIds.map(id => playlist.users[id])

                return (
                  <PlaylistListElem
                    key={track.id}
                    track={track}
                    songPlaying={playlist.song_playing}
                    usersForTrack={users}
                    updateSongCallback={updateSongCallback}/>
                )
              })
            }

            {divider}
          </div>
        )
      })
  }

  let player = (
    <ReactAudioPlayer
      src={playlist.song_playing}
      autoPlay
    />
  )

  let addButton;

  if (trackTotalCount !== 0) {

    if (playlist.creating_playlist) {
      addButton = (
        <Button variant="warning" size="lg" className="mb-4" disabled>
          <Spinner animation="border" className="mr-2"/> Creating playlist
        </Button>
      )

    } else if (!isEmpty(playlist.new_playlist[playlist.minSharedCount])) {
      let url = "#"
      let spotifyUrl = playlist.new_playlist[playlist.minSharedCount].spotify_url

      if (spotifyUrl) {
        url = spotifyUrl
      }

      addButton = (
        (
          <Button variant="success" size="lg" className="mb-4" target="_blank" href={url}>
            Go to my new playlist ➡️
          </Button>
        )
      )

    } else {
      addButton = (
        (
          <Button variant="outline-success" size="lg" className="mb-4" onClick={showModal}>
            Add to my playlists
          </Button>
        )
      )
    }
  }

  let slider;
  let sliderHelp;

  // Ugly slider but it does the job
  if (roomId && playlist.minSharedCountLimit < playlist.maxSharedCountLimit) {
    sliderHelp = (
      <p className={styles.slider_help + " mb-5 mt-3 ml-3 mr-3 text-center"}>
        Select the minimum number of friends that <br/>
        must have the song in their spotify library among the group for it to appear
      </p>
    )

    let current = playlist.minSharedCount
    let min = playlist.minSharedCountLimit
    let max = playlist.maxSharedCountLimit

    slider = (
      <Range
        step={1}
        min={min}
        max={max}
        values={[current]}
        onChange={(values) => {
          setPlaylist(prevState => {
            return {
              ...prevState,
              minSharedCount: values[0]
            }
          })
        }}
        renderTrack={({ props, children }) => (
          <div
            onMouseDown={props.onMouseDown}
            onTouchStart={props.onTouchStart}
            className={styles.tracker}
            style={{
              ...props.style,
            }}
          >
            <div
              ref={props.ref}
              style={{
                height: '5px',
                width: '100%',
                borderRadius: '4px',
                background: getTrackBackground({
                  values: [current],
                  colors: ['#28a745', '#cccccc'],
                  min: min,
                  max: max
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
              {current}
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
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className="text-center mt-3 mb-3">{playlist.name}</h1>
        {slider}
        {sliderHelp}
        {info}
        {addButton}
        {music}
        {player}
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <Toast/>

      <CustomModal
        show={playlist.showConfirmationModal}
        body={"You are creating a playlist on your account, do you wish to continue?"}
        secondaryActionName={"Cancel"}
        secondaryAction={hideModal}
        onHideAction={hideModal}
        primaryActionName={"Add playlist"}
        primaryAction={addPlaylist}
      />
    </div>
  )
}