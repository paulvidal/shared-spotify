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
  });

  const refresh = () => {
    // Do not refresh anything if no roomId exists
    if (!roomId || !playlistId) {
      return null;
    }

    axiosClient.get(getUrl('/rooms/' + roomId + '/playlists/' + playlistId))
      .then(resp => {
        let playlistReceived = resp.data.tracks_per_shared_count

        let minSharedCountLimit = min(Object.keys(playlistReceived).map(i => {
          return parseInt(i)
        }))
        let maxSharedCountLimit = max(Object.keys(playlistReceived).map(i => {
          return parseInt(i)
        }))

        setPlaylist(prevState => {
          return {
            ...prevState,
            ...resp.data,
            minSharedCount: maxSharedCountLimit,
            minSharedCountLimit: minSharedCountLimit,
            maxSharedCountLimit: maxSharedCountLimit
          }
        })
      })
      .catch(error => {
        showErrorToastWithError("Failed to get playlist " + playlistId, error)
      })
  }

  useEffect(refresh, [roomId, playlistId])

  const addPlaylist = () => {
    let confirmation = confirm("You are creating a playlist on your account, do you wish to continue?")

    if (!confirmation) {
      return
    }

    setPlaylist(prevState => {
      return {
        ...prevState,
        creating_playlist: true,
      }
    })

    axiosClient.post(getUrl('/rooms/' + roomId + '/playlists/' + playlistId + '/add'), {
      min_shared_count: playlist.minSharedCount
    }).then(resp => {
        const playlistName = resp.data.name

        setPlaylist(prevState => {
          return {
            ...prevState,
            creating_playlist: false,
            new_playlist: resp.data
          }
        })
        showSuccessToast(`Successfully created in spotify playlist "${playlistName}"`)
      })
      .catch(error => {
        showErrorToastWithError("Failed to create playlist in spotify", error)
      })
      .finally(() => {
        setPlaylist(prevState => {
          return {
            ...prevState,
            creating_playlist: false,
          }
        })
      })
  }

  const updateSongCallback = (song) => {
    setPlaylist(prevState => {
      return {
        ...prevState,
        song_playing: song
      }
    })
  }

  let tracksPerSharedCount = Object.keys(playlist.tracks_per_shared_count)
    .filter(sharedCount => parseInt(sharedCount) >= playlist.minSharedCount)
    .sort()
    .reverse()

  let info;

  let music = (
    <h4 className="mt-5 text-center">No tracks in common found... 😞</h4>
  );

  if (tracksPerSharedCount.length !== 0) {
    let totalCount = sum(
      tracksPerSharedCount
        .map(sharedCount => playlist.tracks_per_shared_count[sharedCount].length)
    )

    info = [
      <p key="count" className="font-weight-bold text-center mb-0">
        {totalCount} songs in common 🎉
      </p>,
      <p key="info" className="font-weight-normal">
        (shared between at least {playlist.minSharedCount} friends)
      </p>
    ]

    music = tracksPerSharedCount
      .map((sharedCount, index) => {
        let divider;

        if (index !== tracksPerSharedCount.length - 1) {
          divider = (
            <div className={styles.group_divider + " mt-5 col-5 col-md-3"}/>
          )
        }

        return (
          <div key={sharedCount} className={styles.common_songs_group}>
            <h5 className="mt-3 mb-3">Songs shared by {sharedCount} friends</h5>
            {playlist.tracks_per_shared_count[sharedCount]
              .sort((track1, track2) => {
                return getArtistsFromTrack(track1).localeCompare(getArtistsFromTrack(track2))
              })
              .map(track => {
                return (
                  <PlaylistListElem
                    key={track.id}
                    track={track}
                    songPlaying={playlist.song_playing}
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

  if (!isEmpty(playlist.tracks_per_shared_count)) {
    if (playlist.creating_playlist) {
      addButton = (
        <Button variant="warning" size="lg" className="mb-4" disabled>
          <Spinner animation="border" className="mr-2"/> Creating playlist
        </Button>
      )

    } else if (!isEmpty(playlist.new_playlist)) {
      let url = "#"

      if (playlist.new_playlist.spotify_url) {
        url = playlist.new_playlist.spotify_url
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
          <OverlayTrigger
            key="top"
            placement="top"
            overlay={
              <Tooltip id={`tooltip-top`}>
                Playlist will be created in spotify and added to your playlists
              </Tooltip>
            }
          >
            <Button variant="outline-success" size="lg" className="mb-4" onClick={addPlaylist}>
              Add to my playlists
            </Button>
          </OverlayTrigger>
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
        <h1 className="text-center mt-3 mb-3">{playlist.type}</h1>
        <p>Room #{roomId}</p>
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
    </div>
  )
}