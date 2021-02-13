import {useEffect, useState} from 'react';
import axios from 'axios';
import Button from 'react-bootstrap/Button';
import styles from '../../styles/login/Login.module.scss'
import CustomHead from "../../components/Head";
import Header from "../../components/Header";
import {useRouter} from "next/router";
import {encodeParams, getUrl} from "../../utils/urlUtils";
import jwt_decode from "jwt-decode";
import setState from "../../utils/stateUtils";
import {showErrorToastWithError} from "../../components/toast";
import LoaderScreen from "../../components/LoaderScreen";
import Footer from "../../components/Footer";

export default function Login() {
  const router = useRouter()
  const { redirect_uri } = router.query

  let redirectUri = "";

  if (redirect_uri) {
    redirectUri = redirect_uri;
  }

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

  const signInSpotify = () => {
    const params = {
      redirect_uri: redirectUri
    }

    window.location.assign(getUrl('/login?' + encodeParams(params)))
  }

  const sendAppleTokens = () => {
    const params = {
      user_id: login.userId,
      user_email: login.userEmail,
      user_name: login.userName,
      musickit_token: login.musicKitToken,
      musickit_user_token: login.musicKitUserToken,
      redirect_uri: redirectUri
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
      showErrorToastWithError("Sign in with apple failed", err, router)
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
      showErrorToastWithError("Apple music sign in failed", err, router)
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

  // we show spotify and apple music buttons
  let buttons = (
    <div className="d-flex flex-column">
      <Button variant="success" size="lg" className={styles.login_button + " mt-5"} onClick={signInSpotify}>
        <div>
          <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo_spotify} />
          <span className={styles.connect_spotify_text}>Connect with Spotify</span>
        </div>
      </Button>

      <Button variant="dark" size="lg" className={styles.login_button + " mt-2"} onClick={signInApple}>
        <div>
          <img src="/apple.svg" alt="Spotify Logo" className={styles.logo_apple} />
          <span className={styles.connect_apple_text}>Connect with Apple Music</span>
        </div>
      </Button>
    </div>
  )

  // this is the case where we login using apple, as we need to do 2 sign in (apple and then apple music)
  if (login.userId && !login.musicKitUserToken) {

    // this button is a copy of the one above, make sure to keep it in sync
    buttons = [
      <p className="mt-5 text-center ml-2 mr-2">
        Thank you for signing in with Apple, we now need to link your account to Apple Music
      </p>,

      <Button size="lg" className={styles.login_button_apple_music + " mt-3"} onClick={signInMusicKit}>
        <div>
          <img src="/applemusic.svg" alt="Apple music logo" className={styles.logo_apple} />
          <span className={styles.connect_apple_text}>Link Apple Music Account</span>
        </div>
      </Button>
    ]

    // Sign in with apple music straight when script are loaded
    document.addEventListener("musickitloaded", signInMusicKit)

  } else if (login.userId && login.musicKitUserToken) {
    buttons = (
      <Button variant="success" size="lg" className="mt-5" onClick={sendAppleTokens}>
        Share music ➡️
      </Button>
    )

    // Once all is good, send tokens and redirect
    sendAppleTokens()
  }

  return (
    <div className={styles.container}>
      <CustomHead>
        <script type="text/javascript" src="https://appleid.cdn-apple.com/appleauth/static/jsapi/appleid/1/en_US/appleid.auth.js" async></script>
        <script type="text/javascript" src="https://js-cdn.music.apple.com/musickit/v1/musickit.js" async></script>
      </CustomHead>

      <Header />

      <main className={styles.main}>
        <h1 className={styles.title}>
          Login
        </h1>

        {buttons}
      </main>

      <Footer/>
    </div>
  )
}
