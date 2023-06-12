"use client";

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

const LoginPage = () => {
    const router = useRouter();
    const authUrl = 'https://accounts.spotify.com/authorize';
    const scope = 'user-read-currently-playing user-read-playback-state';
    const clientId = process.env.NEXT_PUBLIC_SPOTIFY_CLIENT_ID || '';
    const redirectUri = process.env.NEXT_PUBLIC_SPOTIFY_REDIRECT_URI || '';
  
    useEffect(() => {
      const params = new URLSearchParams({
        client_id: clientId,
        response_type: 'code',
        redirect_uri: redirectUri,
        scope: scope,
      });
  
      router.push(`${authUrl}?${params.toString()}`);
    }, []);
  
    return null;
  };
  
  export default LoginPage;