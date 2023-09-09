"use client";

import { useRouter , useSearchParams} from 'next/navigation';
import { useEffect } from 'react';

const CallbackPage = () => {
  const router = useRouter();
	const query = useSearchParams()
	const code = query.get('code') as string
  const tokenUrl = 'https://accounts.spotify.com/api/token';
  const redirectUri = process.env.NEXT_PUBLIC_SPOTIFY_REDIRECT_URI_NOTE || '';
  const clientId = process.env.NEXT_PUBLIC_SPOTIFY_CLIENT_ID || '';
  const clientSecret = process.env.NEXT_PUBLIC_SPOTIFY_CLIENT_SECRET || '';


  useEffect(() => {
    const fetchData = async () => {
      const params = new URLSearchParams();
      params.append('grant_type', 'authorization_code');
      params.append('code', code);
      params.append('redirect_uri', redirectUri);
      params.append('client_id', clientId);
      params.append('client_secret', clientSecret);

      const response = await fetch(tokenUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: params.toString(),
      });

      const data = await response.json();
      const accessToken = data.access_token;
			const refreshToken = data.refresh_token;

      // アクセストークンをセッションストレージに保存する
      sessionStorage.setItem('access_token', accessToken);
			sessionStorage.setItem('refresh_token', refreshToken);

      // リダイレクト
      router.push('/note/home');
    };

    fetchData();
  }, [router]);

  return null; // ページコンポーネントのレンダリングが不要なため、nullを返します
};

export default CallbackPage;
