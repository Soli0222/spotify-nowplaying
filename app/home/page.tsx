"use client";

import { useRouter } from 'next/navigation';
import { useState ,useEffect } from 'react';

type Artist = {
  name: string;
};
type TrackData = {
  track_name: string;
  track_url: string;
  artist_name: string;
  track_enc: string;
}
type ShareURL = {
  psr_track: string;
}

const HomePage = () => {
  const [trackData, setTrackData] = useState<TrackData | null>(null);
  const [shareURL, setShareURL] = useState<ShareURL | null>(null);
  const [error, setError] = useState<string | null>(null); // エラーメッセージの状態を追加
  const router = useRouter();
  useEffect(() => {
    const fetchData = async () => {
      const access_token = sessionStorage.getItem('access_token') || '';
      const headers = {
        Authorization: `Bearer ${access_token}`,
        'Accept-Language': 'ja',
      };

      const response = await fetch('https://api.spotify.com/v1/me/player?market=JP', { headers });

      if (response.status !== 200) {
        let statustext
        if (response.status === 204) {
          statustext = "曲が再生されていません。曲を再生してからリロードしてください。" 
        }
        else {
          statustext = "にゃんらかのエラーです"
        }
        // ステータスコードが200以外の場合、エラーメッセージを表示して処理を中断
        setError(`API Error: StatusCode ${response.status} - ${statustext}`);
        return;
      }

      const data = await response.json();
      //ここからトラックデータ系の処理
      if (data["currently_playing_type"] == "track") {
        const TrackArtists: Artist[] = data['item']['artists'];
        const TrackArtist = TrackArtists.map(artist => artist['name']).join(', ');

        const fetchedData: TrackData = {
          track_name: data['item']['name'],
          track_url: data['item']['external_urls']['spotify'],
          artist_name: TrackArtist,
          track_enc: encodeURIComponent(`${data['item']['name']} / ${TrackArtist}\n#NowPlaying #PsrPlaying`),
        }
        setTrackData(fetchedData);

        const shareURLData: ShareURL = {
          psr_track: `https://mi.soli0222.com/share?url=${fetchedData.track_url}&text=${fetchedData.track_enc}`
        };
        setShareURL(shareURLData);
      }
      else if (data["currently_playing_type"] == "episode") {
        const response = await fetch('https://api.spotify.com/v1/me/player/currently-playing?market=JP&additional_types=episode', { headers });
        const data = await response.json();
        //console.log(data)

        const fetchedData: TrackData = {
          track_name: data['item']['name'],
          track_url: data['item']['external_urls']['spotify'],
          artist_name: data['item']['show']['name'],
          track_enc: encodeURIComponent(`${data['item']['name']} / ${data['item']['show']['name']}\n#NowPlaying`),
        }
        setTrackData(fetchedData);

        const shareURLData: ShareURL = {
          psr_track: `https://mi.soli0222.com/share?url=${fetchedData.track_url}&text=${fetchedData.track_enc}`,
        };
        setShareURL(shareURLData);
      }
    };
    
    fetchData();
  },[]);

  useEffect(() => {
    if (shareURL) {
      router.push(shareURL.psr_track);
    }
  }, [shareURL]);
  
  return (
    <div>
      {error && <p>{error}</p>}
    </div>
  )
};

export default HomePage;
