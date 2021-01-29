import {useEffect, useState} from 'react';
import axios from 'axios';
import {Button} from 'react-bootstrap';
import styles from '../../styles/Home.module.scss'
import {getUrl, encodeParams} from "../../utils/urlUtils";
import CustomHead from "../../components/Head";
import Header from "../../components/Header";
import {useRouter} from "next/router";
import setState from "../../utils/stateUtils";
import LoaderScreen from "../../components/LoaderScreen";
import jwt_decode from "jwt-decode";
import {showErrorToastWithError} from "../../components/toast";

export default function Home() {
  const router = useRouter()
  const axiosClient = axios.create({
    withCredentials: true
  })

  const [login, setLogin] = useState({
    loading: true,
    userId: null,
    userEmail: null,
    userName: null,
    musicKitToken: process.env.NEXT_PUBLIC_MUSICKIT_TOKEN,
    musicKitUserToken: null
  });

  const sendTokens = () => {
    const params = {
      user_id: login.userId,
      user_email: login.userEmail,
      user_name: login.userName,
      musickit_token: login.musicKitToken,
      musickit_user_token: login.musicKitUserToken,
      redirect_url: "" // TODO: specify for when we want to redirect
    };

    // Redirect to the server for the redirect
    window.location.assign(getUrl('/callback/apple?' + encodeParams(params)))
  }

  const signInApple = () => {
    AppleID.auth.init({
      clientId : 'com.sharedspotify.apple.login',
      scope : 'name',
      redirectURI : process.env.NEXT_PUBLIC_APPLE_LOGIN_REDIRECT_URL,
      state : window.location.href,
      usePopup : true
    });

    AppleID.auth.signIn().then(response => {
      console.log("response sign in")
      console.log(response)

      let decoded = jwt_decode(response.authorization.id_token)
      console.log(decoded)

      let name = "";

      if (response.user && response.user.name) {
        name = response.user.name.firstName + " " + response.user.name.lastName
      }

      setState(setLogin, {
        userId: decoded.sub,
        userEmail: decoded.email,
        userName: name
      })

      signInMusicKit()

    }).catch(err => {
      showErrorToastWithError("Sign in with apple failed", err)
      console.error(err);
    });
  };

  const signInMusicKit = () => {
    let musicKit = MusicKit.configure({
      developerToken: login.musicKitToken,
      app: {
        name: 'Shared spotify',
        build: '1'
      }
    });

    console.log("apple music sign in")

    musicKit.authorize().then(musicUserToken => {
      console.log("Authorised music kit");
      console.log(musicUserToken);

      setState(setLogin, {
        musicKitUserToken: musicUserToken
      })

    }).catch((err) => {
      showErrorToastWithError("Apple music sign in failed", err)
      console.error(err)
    })
  }

  const refresh = () => {
    axiosClient.get(getUrl('/user'))
      .then(resp => {
        router.push('/rooms')
      })
      .catch(error => {
        setState(setLogin, {loading: false})
      })
  }

  useEffect(refresh, [])

  // Use a loader screen if nothing is ready
  if (login.loading) {
    return (
      <LoaderScreen/>
    )
  }

  let button;

  if (!login.userId) {
    button = (
      <Button variant="outline-success" size="lg" className="mt-3" onClick={signInApple}>
        Login to Apple
      </Button>
    )

    // Sign in with apple straight when script are loaded
    document.addEventListener("musickitloaded", signInApple)

  } else if (!login.musicKitUserToken) {
    button = (
      <Button variant="outline-success" size="lg" className="mt-3" onClick={signInMusicKit}>
        Connect Apple music account
      </Button>
    )

    // Sign in with apple music straight when script are loaded
    document.addEventListener("musickitloaded", signInMusicKit)

  } else {
    button = (
      <Button variant="outline-success" size="lg" className="mt-3" onClick={sendTokens}>
        All good !
      </Button>
    )

    // Once all is good, send tokens and redirect
    sendTokens()
  }

  return (
    <div className={styles.container}>
      <CustomHead>
        <script type="text/javascript" src="https://appleid.cdn-apple.com/appleauth/static/jsapi/appleid/1/en_US/appleid.auth.js"></script>
        <script type="text/javascript" src="https://js-cdn.music.apple.com/musickit/v1/musickit.js"></script>
      </CustomHead>

      <Header />

      <main className={styles.main}>
        <h1>Sign in with Apple</h1>

        <p className="mt-4 text-center">
          Please click on the button if you are not redirected automatically to login
        </p>

        { button }
      </main>

      <footer className={styles.footer}>
        Powered by{' '}
        <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      </footer>
    </div>
  )
}
