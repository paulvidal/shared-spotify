import styles from "../styles/Home.module.scss";
import Link from "next/link";
import {Image} from "react-bootstrap";

export default function Header() {
  return (
    <header className={styles.header}>
      <Link href="/">
        <a>
          <Image src="/logo.png" className={styles.header_logo}/>
        </a>
      </Link>
    </header>
  )
}