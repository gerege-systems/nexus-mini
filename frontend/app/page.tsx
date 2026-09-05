'use client';

import { useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Badge, Button, Card, Icons } from '@gerege-systems/ui';
import { MktHeader, MktFooter } from '@/components/mkt';
import { useT } from '@/lib/i18n';
import { setupStatus } from '@/lib/setup';

// Нүүр — ерөнхий танилцуулга. Апп дэлгүүр (/apps) ба модуль хөгжүүлэх
// (/developers) тус тусдаа хуудсуудтай.
export default function Landing() {
  const { t } = useT();
  const router = useRouter();
  // Платформ админгүй суулгац дээр танилцуулга худал — шидтэн рүү.
  useEffect(() => {
    void setupStatus().then((s) => s.required && router.replace('/setup'));
  }, [router]);

  return (
    <div className="flex min-h-dvh flex-col">
      <MktHeader />

      <main className="flex-1">
        {/* hero */}
        <section className="mx-auto grid max-w-6xl items-center gap-10 px-4 py-14 md:px-6 md:py-20 lg:grid-cols-2">
          <div>
            <Badge tone="accent" variant="outline" dot>
              v1.1 · Apache 2.0
            </Badge>
            <h1 className="text-foreground mt-4 text-3xl font-semibold text-balance md:text-4xl">
              {t('Үйл ажиллагааны нэгдсэн дижитал платформ')}
            </h1>
            <p className="text-foreground-muted mt-4 text-base text-pretty">
              {t(
                'Байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэг дор холбодог модульт платформ — цөм нь суурийг, апп дэлгүүр нь боломжуудыг өгнө.',
              )}
            </p>
            <div className="mt-6 flex flex-wrap gap-3">
              <Button size="xl" asChild>
                <Link href="/signup">{t('Байгууллагаа бүртгүүлэх')}</Link>
              </Button>
              <Button size="xl" variant="secondary" asChild>
                <Link href="/developers" className="font-mono">
                  $ docker compose up
                </Link>
              </Button>
            </div>
          </div>

          <Terminal />
        </section>

        {/* цөмийн 3 давуу тал */}
        <section id="core" className="mx-auto max-w-6xl px-4 pb-14 md:px-6">
          <div className="grid gap-4 md:grid-cols-3">
            <Feature
              tag="tenant + RLS"
              title={t('Тусгаарлалт DB давхаргад')}
              body={t(
                'Байгууллага бүрийн өгөгдөл PostgreSQL Row-Level Security-ээр тусгаарлагдана — кодын алдаа ч хана даван харагдуулахгүй.',
              )}
            />
            <Feature
              tag="rbac + audit"
              title={t('Эрх тунхаглалаар, бүртгэл гинжээр')}
              body={t(
                'Permission суулгах үед role-уудад автоматаар оноогдоно; бүх чухал үйлдэл hash chain-тэй audit бүртгэлд үлдэнэ.',
              )}
            />
            <Feature
              tag="nexus.Module"
              title={t('9 метод = таны модуль')}
              body={t(
                'Go interface хэрэгжүүлээд каталогт PR илгээхэд л таны модуль store-д — нэг бинари, микросервисийн төвөггүй.',
              )}
            />
          </div>
        </section>

        {/* хоёр гарц */}
        <section className="mx-auto max-w-6xl px-4 pb-14 md:px-6">
          <div className="grid gap-4 md:grid-cols-2">
            <NavCard
              href="/apps"
              icon={<Icons.ShoppingCart className="size-5" />}
              title={t('Апп дэлгүүр')}
              body={t(
                'Байгууллага бүр өөрт хэрэгтэй модулиа сонгож суулгана — суусан апп эрх, цэсээ өөрөө авчирна. Одоо байгаа аппуудыг тайлбартай нь үзэх.',
              )}
            />
            <NavCard
              href="/developers"
              icon={<Icons.Zap className="size-5" />}
              title={t('Модуль хөгжүүлэх')}
              body={t(
                'Модуль бол есөн метод хэрэгжүүлсэн Go package. Файлын бүтэц, permission, миграц, route — бүрэн гарын авлага.',
              )}
            />
          </div>
        </section>

        {/* CTA */}
        <section className="mx-auto max-w-6xl px-4 pb-8 md:px-6">
          <Card className="flex flex-wrap items-center gap-4">
            <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md">
              <Icons.Package className="size-5" aria-hidden />
            </span>
            <div className="min-w-[15rem] flex-1">
              <p className="text-foreground font-medium">{t('Өөрөө ажиллуулж үзэх үү?')}</p>
              <p className="text-foreground-muted text-sm">
                <code className="font-mono text-xs">git clone</code> → {t('env-ээ бөглөөд')}{' '}
                <code className="font-mono text-xs">make migrate</code> →{' '}
                <code className="font-mono text-xs">make serve</code>. {t('Эсвэл')}{' '}
                <code className="font-mono text-xs">docker compose up</code>.
              </p>
            </div>
            <Button asChild>
              <Link href="/signup">{t('Эсвэл эндээ бүртгүүлэх')}</Link>
            </Button>
          </Card>
        </section>
      </main>

      <MktFooter />
    </div>
  );
}

function Feature({ tag, title, body }: { tag: string; title: string; body: string }) {
  return (
    <Card>
      <code className="text-foreground-subtle font-mono text-xs">{tag}</code>
      <h2 className="text-foreground mt-2 font-semibold">{title}</h2>
      <p className="text-foreground-muted mt-2 text-sm">{body}</p>
    </Card>
  );
}

function NavCard({
  href,
  icon,
  title,
  body,
}: {
  href: string;
  icon: React.ReactNode;
  title: string;
  body: string;
}) {
  return (
    <Card variant="interactive" asChild>
      <Link href={href}>
        <span className="bg-background-muted text-foreground-muted mb-3 grid size-10 place-items-center rounded-md">
          {icon}
        </span>
        <h2 className="text-foreground flex items-center gap-1.5 font-semibold">
          {title}
          <Icons.ArrowRight className="size-4" aria-hidden />
        </h2>
        <p className="text-foreground-muted mt-2 text-sm">{body}</p>
      </Link>
    </Card>
  );
}

/** Чимэглэл — дэлгэц уншигчид хэрэггүй тул бүхэлд нь aria-hidden. */
function Terminal() {
  const { t } = useT();
  return (
    <div
      aria-hidden
      className="border-border bg-background-subtle overflow-hidden rounded-lg border font-mono text-xs"
    >
      <div className="border-border bg-background-muted flex items-center gap-1.5 border-b px-3 py-2">
        <span className="size-2.5 rounded-full bg-[#f87171]" />
        <span className="size-2.5 rounded-full bg-[#fbbf24]" />
        <span className="size-2.5 rounded-full bg-[#34d399]" />
        <span className="text-foreground-subtle ml-2 truncate">nexus-mini — zsh</span>
      </div>
      <div className="text-foreground space-y-1 overflow-x-auto p-4 leading-relaxed">
        <div>
          <span className="text-accent">$</span> git clone gerege-systems/nexus-mini
        </div>
        <div>
          <span className="text-accent">$</span> make migrate
        </div>
        <div className="text-foreground-muted">
          {t('миграц: цөм ok · devices ok · organisation ok')}
        </div>
        <div className="text-foreground-muted">✓ {t('платформын админ үүслээ')}</div>
        <div>
          <span className="text-accent">$</span> make serve
        </div>
        <div className="text-foreground-muted">
          nexus-mini API :8084 (2 {t('модуль')})
        </div>
        <div>
          <span className="text-accent">$</span>{' '}
          <span className="bg-foreground/70 inline-block h-3.5 w-2 align-middle" />
        </div>
      </div>
      <div className="border-border text-foreground-subtle flex flex-wrap gap-x-4 gap-y-1 border-t px-4 py-2">
        <span>Go 1.25</span>
        <span>PostgreSQL 16 · RLS</span>
        <span>Next.js</span>
        <span>{t('нэг бинари')}</span>
      </div>
    </div>
  );
}
