import {useEffect, useState} from 'react';
import axios from 'axios';
import {Button} from 'react-bootstrap';
import Link from 'next/link'
import styles from '../styles/Home.module.scss'
import {isEmpty} from "lodash"
import {getUrl} from "../utils/urlUtils";
import CustomHead from "../components/Head";
import Header from "../components/Header";

export default function Home() {
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [userInfos, setUserInfos] = useState({});

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => setUserInfos(resp.data))
      .catch(error => {})
  }

  useEffect(refresh, [])

  let greetings;

  if (!isEmpty(userInfos)) {
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
      <CustomHead />

      <Header />

      {greetings}

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>
    </div>
  )
}
