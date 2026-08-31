import type { Metadata } from 'next';
import { headers } from 'next/headers';
import { Geist, Geist_Mono } from 'next/font/google';
import { DesignSystemProvider, Toaster, TooltipProvider, mnStrings } from '@craftzbay/ui';
import './globals.css';

const sans = Geist({
  subsets: ['latin', 'cyrillic', 'cyrillic-ext'],
  weight: ['400', '500', '600'],
  variable: '--font-geist-sans',
  display: 'swap',
});

const mono = Geist_Mono({
  subsets: ['latin', 'cyrillic-ext'],
  weight: ['400', '500'],
  variable: '--font-geist-mono',
  display: 'swap',
});

export const metadata: Metadata = {
  title: { default: 'nexus-mini · Платформын админ', template: '%s · Платформын админ' },
  description: 'nexus-mini платформын удирдлагын систем',
};

// Эхний зурагдалтаас өмнө класс тавина — theme flash-аас сэргийлнэ.
// Сан `dark` классаар ажиллана (data-theme биш).
const themeScript = `
try {
  var t = localStorage.getItem('nexus_theme');
  if (t === 'dark' || (t !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches)) {
    document.documentElement.classList.add('dark');
  }
} catch (e) {}
`;

// middleware хүсэлт бүрт CSP nonce үүсгэдэг — доорх inline script түүнийг
// авахгүй бол хатуу script-src-д хаагдана.
export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const nonce = (await headers()).get('x-nonce') ?? undefined;

  return (
    <html lang="mn" suppressHydrationWarning className={`${sans.variable} ${mono.variable}`}>
      <head>
        <script nonce={nonce} dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="bg-background text-foreground font-sans antialiased">
        <DesignSystemProvider strings={mnStrings}>
          <TooltipProvider>
            {children}
            <Toaster />
          </TooltipProvider>
        </DesignSystemProvider>
      </body>
    </html>
  );
}
