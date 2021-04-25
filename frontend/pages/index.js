import {useEffect, useState} from 'react';
import axios from 'axios';
import {Button} from 'react-bootstrap';
import styles from '../styles/Home.module.scss'
import {getUrl} from "../utils/urlUtils";
import CustomHead from "../components/Head";
import Header from "../components/Header";
import {useRouter} from "next/router";
import setState from "../utils/stateUtils";
import LoaderScreen from "../components/LoaderScreen";
import {faAngleDown} from "@fortawesome/free-solid-svg-icons";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {animateScroll as scroll} from 'react-scroll'

export default function Home() {
  const router = useRouter()
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [home, setHome] = useState({
    login: false,
    loading: true,
  });

  let scrollBottom = () => {
    // 1.2 is to make sure really go to the bottom of the page
    scroll.scrollTo(document.body.scrollHeight * 1.2, {
      duration: 2000,
      smooth: true,
    });
  }

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => {
        setState(setHome, {
          login: true,
          loading: false
        })
      })
      .catch(error => {
        setState(setHome, {
          login: false,
          loading: false
        })
      })
  }

  useEffect(refresh, [])

  // Use a loader screen if nothing is ready
  if (home.loading) {
    return (
      <LoaderScreen/>
    )
  }

  const nextPage = home.login ? '/rooms' : '/login'

  let button = (
    <Button variant="success" size="lg" className="mt-2" onClick={() => router.push(nextPage)}>
      Connect music account
    </Button>
  )

  if (home.login) {
    button = (
      <Button variant="success" size="lg" className="mt-2" onClick={() => router.push(nextPage)}>
        Share music now
      </Button>
    )
  }

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <p className="mt-4 text-center ml-2 mr-2">
          The best place to find and share common songs between friends
        </p>

        {button}

        <FontAwesomeIcon icon={faAngleDown} className={styles.angle} onClick={scrollBottom} />
      </main>

      <main className={styles.main_2}>
        <h1 className={styles.presentation_title}>How it works</h1>
        <div className="container">
          <div className={"mt-5 row " + styles.presentation}>
            <div className={styles.presentation_panel + " " + styles.presentation_panel_1 + " col-md-4 col-12"}>
              <img src="/share.svg" alt="Spotify Logo" className={styles.presentation_image} />
              <h5 className={styles.presentation_header}>1. Create a room and share it with your friends</h5>
              <p className={styles.presentation_text}>
                Send the room link to all your friends so they can easily join in 1 click.
              </p>
            </div>
            <div className={styles.presentation_panel + " " + styles.presentation_panel_2 + " col-md-4 col-12"}>
              <img src="/social.svg" alt="Spotify Logo" className={styles.presentation_image} />
              <h5 className={styles.presentation_header}>2. Discover common songs</h5>
              <p className={styles.presentation_text}>
                Thanks to a powerful algorithm, discover all your common songs compiled in various playlists.
              </p>
            </div>
            <div className={styles.presentation_panel + " " + styles.presentation_panel_3 + " col-md-4 col-12"}>
              <img src="/playlist.svg" alt="Spotify Logo" className={styles.presentation_image} />
              <h5 className={styles.presentation_header}>3. Add generated playlists to your favourite music app</h5>
              <p className={styles.presentation_text}>
                Create the playlists you want in 1 click, so you can listen to them from your music app!
              </p>
            </div>
          </div>
        </div>

        <Button variant="success" size="lg" className={styles.presentation_button + " text-center mt-5"} onClick={() => router.push(nextPage)}>
          Join the fun
        </Button>
      </main>

      {/* dummy tag to add the space lost because of the footer */}
      <div className={styles.dummy}/>

      <footer className={styles.footer}>
        Powered by
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
        {'and'}
        <img src="/applemusic.svg" alt="Apple music Logo" className={styles.logo_apple_music} />
      </footer>
    </div>
  )
}
