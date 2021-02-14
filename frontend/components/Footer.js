import styles from "../styles/HeaderFooter.module.scss";

export default function Footer(props) {
  return (
    <footer className={styles.footer}>
      Powered by
      <img src="/spotify.svg" alt="Spotify Logo" className={styles.logo} />
      {'and'}
      <img src="/applemusic.svg" alt="Apple music Logo" className={styles.logo_apple_music} />
    </footer>
  )
}