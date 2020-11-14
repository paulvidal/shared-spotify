import {useEffect, useState} from 'react';
import axios from 'axios';
import {Image, Button, Row} from 'react-bootstrap';
import Link from 'next/link'
import styles from '../styles/Home.module.scss'
import {isEmpty} from "lodash"
import {getUrl} from "../utils/urlUtils";
import CustomHead from "../components/Head";
import Header from "../components/Header";
import {useRouter} from "next/router";
import setState from "../utils/stateUtils";
import LoaderScreen from "../components/LoaderScreen";

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

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <Button href={getUrl('/login')} variant="outline-success" size="lg" className="mt-5">
          Connect spotify account
        </Button>
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>
    </div>
  )
}
