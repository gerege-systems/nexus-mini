import type { Metadata } from 'next';

export const metadata: Metadata = { title: 'Байгууллага нэмэх' };

// Хуудас нь client component тул metadata-г зөвхөн layout-аас өгнө.
export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
