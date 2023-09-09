"use client";

import { useRouter } from 'next/navigation';
import { useEffect } from 'react';

export default function Index() {
  const router = useRouter();

  useEffect(() => {
    // ページがアクセスされた瞬間に/loginに遷移
    router.push('/tweet/login');
  }, []);

  return null;
}
