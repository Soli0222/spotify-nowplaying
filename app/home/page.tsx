"use client";

import { useState ,useEffect } from 'react';

type Artist = {
  name: string;
};
type TrackData = {
  track_name: string;
  track_url: string;
  artist_name: string;
  album_name: string;
  album_artist: string;
  album_url: string;
  jacket_url: string;
  track_enc: string;
  album_enc: string;
}
type ShareURL = {
  tweet_track: string;
  tweet_album: string;
  psr_track: string;
  psr_album: string;
}

const HomePage = () => {
  const [trackData, setTrackData] = useState<TrackData | null>(null);
  const [shareURL, setShareURL] = useState<ShareURL | null>(null);
  useEffect(() => {
    const fetchData = async () => {
      const access_token = sessionStorage.getItem('access_token') || '';
      const headers = {
        Authorization: `Bearer ${access_token}`,
        'Accept-Language': 'ja',
      };

      const response = await fetch('https://api.spotify.com/v1/me/player?market=JP', { headers });
      const data = await response.json();

      //ここからトラックデータ系の処理
      if (data["currently_playing_type"] == "track") {
        const TrackArtists: Artist[] = data['item']['artists'];
        const TrackArtist = TrackArtists.map(artist => artist['name']).join(', ');
        const Alubmartists: Artist[] = data['item']['album']['artists'];
        const ArackArtist = Alubmartists.map(artist => artist['name']).join(', ');

        const fetchedData: TrackData = {
          track_name: data['item']['name'],
          track_url: data['item']['external_urls']['spotify'],
          artist_name: TrackArtist,
          album_name: data['item']['album']['name'],
          album_artist: ArackArtist,
          album_url: data['item']['album']['external_urls']['spotify'],
          jacket_url: data['item']['album']['images'][0]['url'],
          track_enc: encodeURIComponent(`${data['item']['name']} / ${TrackArtist}\n#NowPlaying`),
          album_enc: encodeURIComponent(`${data['item']['album']['name']} / ${ArackArtist}\n#NowPlaying`),
        }
        setTrackData(fetchedData);
      }
      else if (data["currently_playing_type"] == "episode") {
        const response = await fetch('https://api.spotify.com/v1/me/player/currently-playing?market=JP&additional_types=episode', { headers });
        const data = await response.json();
        //console.log(data)

        const fetchedData: TrackData = {
          track_name: data['item']['name'],
          track_url: data['item']['external_urls']['spotify'],
          artist_name: data['item']['show']['name'],
          album_name: data['item']['show']['name'],
          album_artist: data['item']['show']['publisher'],
          album_url: 'None',
          jacket_url: data['item']['images'][0]['url'],
          track_enc: encodeURIComponent(`${data['item']['name']} / ${data['item']['show']['name']}\n#NowPlaying`),
          album_enc: encodeURIComponent(`${data['item']['show']['name']} / ${data['item']['show']['publisher']}\n#NowPlaying`),
        }
        setTrackData(fetchedData);
      }
      
      const fetchedData: ShareURL ={
        tweet_track: trackData ? `https://twitter.com/intent/tweet?url=${trackData.track_url}&text=${trackData?.track_enc}`: '',
        tweet_album: trackData ? `https://twitter.com/intent/tweet?url=${trackData.album_url}&text=${trackData?.album_enc}`: '',
        psr_track: trackData ? `https://mi.soli0222.com/share?url=${trackData.track_url}&text=${trackData?.track_enc}`: '',
        psr_album: trackData ? `https://mi.soli0222.com/share?url=${trackData.album_url}&text=${trackData?.album_enc}`: '',
      }
      setShareURL(fetchedData);
      
      //const trackEnc = encodeURIComponent(`${trackData.track_name} / ${trackData.artist_name}\n#NowPlaying`);
      
    };

    fetchData();
  },[trackData]);
  
  return (
    <div>
      {/* trackDataが存在する場合にのみ、情報を表示 */}
      {trackData && (
        <div>
          <h2>Track Data:</h2>
          <p>Track Name: {trackData.track_name}</p>
          <p>Track URL: {trackData.track_url}</p>
          <p>Artist Name: {trackData.artist_name}</p>
          <p>Album Name: {trackData.album_name}</p>
          <p>Album Artist: {trackData.album_artist}</p>
          <p>Album URL: {trackData.album_url}</p>
          <p>Jacket URL: {trackData.jacket_url}</p>
          <p>Track Encode: {trackData.track_enc}</p>
          <p>Album Encode: {trackData.album_enc}</p><br></br>
          <a href={shareURL?.tweet_track} target="_blank">Tweet Track</a><br></br>
          <a href={shareURL?.tweet_album} target="_blank">Tweet Alubm</a><br></br>
          <a href={shareURL?.psr_track} target="_blank">Note(Polestar) Track</a><br></br>
          <a href={shareURL?.psr_album} target="_blank">Note(Polestar) Album</a><br></br>
        </div>
      )}
    </div>
  );
};

export default HomePage;
