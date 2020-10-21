import { useState, useEffect } from 'react';
import axios from 'axios';
import Head from 'next/head'
import {Button} from 'react-bootstrap';
import styles from '../styles/Home.module.scss'

import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';

export default function Home() {
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [user, setUser] = useState(null);

  useEffect(() => {
    axiosClient.get('http://localhost:8080/user')
      .then(resp => setUser(resp.data.name))
      .catch(error => {
        let msg = error.response ? error.response.data.message : error;

        toast.error('An error occured: ' + msg, {
          position: "top-right",
          autoClose: 5000,
          hideProgressBar: true,
          closeOnClick: true,
          pauseOnHover: true,
          draggable: true,
          progress: undefined,
        });
      })
  }, [user])

  let greetings;

  if (user) {
    greetings = (
      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <h1 className={styles.name_title}>
          {user}
        </h1>

        <Button variant="outline-success" size="lg" className="mt-5">
          Start sharing music ➡️
        </Button>
      </main>
    )

  } else {
    greetings = (
      <main className={styles.main}>
        <h1 className={styles.title}>
          Welcome to <strong className="text-success">Shared Spotify</strong>
        </h1>

        <Button href="http://localhost:8080/login" variant="outline-success" size="lg" className="mt-5">
          Connect spotify account
        </Button>
      </main>
    )
  }

  return (
    <div className={styles.container}>
      <Head>
        <title>Create Next App</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>

      {greetings}

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>

      <ToastContainer />
    </div>
  )
}
