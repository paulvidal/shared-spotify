import {useEffect, useState} from 'react';
import axios from 'axios';
import Head from 'next/head'
import {Button} from 'react-bootstrap';
import Link from 'next/link'
import styles from '../styles/Home.module.scss'
import _ from "lodash"
import {getUrl} from "../utils/urlUtils";

export default function Home() {
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [userInfos, setUserInfos] = useState({});

  const refresh = () => {
    axiosClient.get(getUrl('/api/user'))
      .then(resp => setUserInfos(resp.data.user_infos))
      .catch(error => {})
  }

  useEffect(refresh, [])

  let greetings;

  if (!_.isEmpty(userInfos)) {
    greetings = (
      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <h1 className={styles.name_title}>
          {userInfos.name}
        </h1>

        <Link href="/rooms">
          <Button variant="outline-success" size="lg" className="mt-5">
            Start sharing music ➡️
          </Button>
        </Link>
      </main>
    )

  } else {
    greetings = (
      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <Button href={getUrl('/login')} variant="outline-success" size="lg" className="mt-5">
          Connect spotify account
        </Button>
      </main>
    )
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Shared Spotify</title>
        <link rel="icon" href="/spotify.svg" />
      </Head>

      {greetings}

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>
    </div>
  )
}
