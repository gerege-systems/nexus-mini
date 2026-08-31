import type { Metadata } from 'next';

export const metadata: Metadata = { title: 'Хэрэглэгчид' };

// Хуудас нь client component тул metadata-г зөвхөн layout-аас өгнө.
export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
