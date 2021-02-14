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
import { Element, Events, animateScroll as scroll, scrollSpy, scroller } from 'react-scroll'

export default function Home() {
  const router = useRouter()
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [home, setHome] = useState({
    loading: true,
  });

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => {
        router.push('/rooms')
      })
      .catch(error => {
        setState(setHome, {loading: false})
      })
  }

  useEffect(refresh, [])

  // Use a loader screen if nothing is ready
  if (home.loading) {
    return (
      <LoaderScreen/>
    )
  }

  let timer;
  let autoscroll = false;
  let lastScrollTop = 0;

  let scrollTop = () => {
    if (autoscroll) {
      return;
    }

    autoscroll = true;

    scroll.scrollTo(0, {
      duration: 1500,
      smooth: true,
    });

    console.log("scroll top")

    autoscroll = false;
  }

  let scrollBottom = () => {
    if (autoscroll) {
      return;
    }

    autoscroll = true;

    scroll.scrollTo(document.body.scrollHeight * 1.2, {
      duration: 2000,
      smooth: true,
    });

    console.log("scroll bottom")

    autoscroll = false;
  }

  document.addEventListener("scroll", function(){
    let st = window.pageYOffset || document.documentElement.scrollTop;
    clearTimeout(timer)

    if (st > lastScrollTop) {
      timer = setTimeout(scrollBottom, 150)

    } else {
      timer = setTimeout(scrollTop, 150)
    }

    lastScrollTop = st <= 0 ? 0 : st; // For Mobile or negative scrolling
    autoscroll = false
  }, false);

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <p className="mt-4 text-center ml-2 mr-2">
          The place to find and share common songs between friends
        </p>

        <Button variant="success" size="lg" className="mt-2" onClick={() => router.push('/login')}>
          Connect music account
        </Button>

        <FontAwesomeIcon icon={faAngleDown} className={styles.angle} onClick={scrollBottom} />
      </main>

      <main className={styles.main}>
        <p>How it works</p>
      </main>

      <footer className={styles.footer}>
        Powered by
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
        {'and'}
        <img src="/applemusic.svg" alt="Apple music Logo" className={styles.logo_apple_music} />
      </footer>
    </div>
  )
}
