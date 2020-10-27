import Head from "next/head";

export default function CustomHead() {
  return (
    <Head>
      <title>Shared Spotify</title>
      <link rel="icon" href="/logo.svg" />
      <meta property="og:title" content="Shared spotify" />
      <meta property="og:description" content="The best way to create playlist with songs multiple like!" />
      <meta property="og:image" content={process.env.NEXT_PUBLIC_URL + "/share.png"} />

      {/* always allow referrer to exist */}
      <meta name="referrer" content="always" />
    </Head>
  )
}