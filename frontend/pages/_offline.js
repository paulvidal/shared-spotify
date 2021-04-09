import styles from '../styles/Home.module.scss'
import CustomHead from "../components/Head";
import Header from "../components/Header";
import Footer from "../components/Footer";

export default function Offline() {

  return (
    <div className={styles.container}>
      <CustomHead />

      <Header />

      <main className={styles.main}>
        <h1 className="mb-2 text-center">Offline</h1>
        <p className="mt-5 text-center font-weight-bold">You don't have any internet access 😞</p>
      </main>

      <Footer/>
    </div>
  )
}