"use client";

import { useEffect } from 'react';

const HomePage = () => {
  useEffect(() => {
    const fetchData = async () => {
      const access_token = sessionStorage.getItem('access_token') || '';
      const headers = {
        Authorization: `Bearer ${access_token}`,
        'Accept-Language': 'ja',
      };

      try {
        const response = await fetch('https://api.spotify.com/v1/me/player?market=JP', { headers });
        const data = await response.json();
        console.log(data);
      } catch (error) {
        console.error(error);
      }
    };

    fetchData();
  }, []);

  return <div>Fetching data...</div>;
};

export default HomePage;
