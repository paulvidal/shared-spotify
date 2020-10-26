import Head from "next/head";

export default function CustomHead() {
  return (
    <Head>
      <title>Shared Spotify</title>
      <link rel="icon" href="/spotify.svg" />
      <meta property="og:title" content="Shared spotify" />
      <meta property="og:description" content="The best way to create playlist with songs multiple likes!" />
      <meta property="og:image" content="/spotify.svg" />
    </Head>
  )
}