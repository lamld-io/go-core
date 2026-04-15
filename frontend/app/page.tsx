import Link from "next/link";

import { checkAuthServiceHealth } from "@/lib/api/client";
import { tryGetAppEnv } from "@/lib/env";

export const dynamic = "force-dynamic";

export default async function Home() {
  const envResult = tryGetAppEnv();
  const healthResult = envResult.ok
    ? await checkAuthServiceHealth({ baseUrl: envResult.env.authApiBaseUrl })
    : { ok: false as const, message: "Missing frontend environment configuration." };

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-8 px-6 py-12 sm:px-10">
      <section className="flex flex-col gap-4 rounded-3xl border border-black/10 bg-white p-8 shadow-sm dark:border-white/10 dark:bg-white/5">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-zinc-500">
          Phase 1 Bootstrap
        </p>
        <div className="space-y-3">
          <h1 className="text-4xl font-semibold tracking-tight text-balance">
            Base Auth Portal frontend is wired and ready.
          </h1>
          <p className="max-w-3xl text-base leading-7 text-zinc-600 dark:text-zinc-300">
            This app is the Next.js bootstrap shell for the current auth service.
            It establishes routing, environment validation, shared API parsing, and
            future entry points for public and protected auth flows.
          </p>
        </div>
      </section>

      <section className="grid gap-6 lg:grid-cols-[1.4fr_1fr]">
        <article className="rounded-3xl border border-black/10 bg-white p-8 shadow-sm dark:border-white/10 dark:bg-white/5">
          <h2 className="text-xl font-semibold">Connectivity</h2>
          <dl className="mt-6 space-y-4 text-sm">
            <div>
              <dt className="text-zinc-500">Frontend env</dt>
              <dd className="mt-1 font-medium">
                {envResult.ok ? "Loaded" : "Configuration required"}
              </dd>
            </div>
            <div>
              <dt className="text-zinc-500">Auth API base URL</dt>
              <dd className="mt-1 break-all font-mono text-xs sm:text-sm">
                {envResult.ok ? envResult.env.authApiBaseUrl : envResult.message}
              </dd>
            </div>
            <div>
              <dt className="text-zinc-500">Health probe</dt>
              <dd className="mt-1 font-medium">
                {healthResult.ok ? "Reachable" : "Unavailable"}
              </dd>
            </div>
            <div>
              <dt className="text-zinc-500">Probe detail</dt>
              <dd className="mt-1 text-zinc-600 dark:text-zinc-300">
                {healthResult.ok
                  ? `${healthResult.data.service} returned ${healthResult.data.status}`
                  : healthResult.message}
              </dd>
            </div>
          </dl>
        </article>

        <aside className="rounded-3xl border border-black/10 bg-white p-8 shadow-sm dark:border-white/10 dark:bg-white/5">
          <h2 className="text-xl font-semibold">Next Routes</h2>
          <div className="mt-6 flex flex-col gap-3 text-sm">
            <Link className="route-link" href="/login">
              Open login placeholder
            </Link>
            <Link className="route-link" href="/profile">
              Open protected placeholder
            </Link>
          </div>
          <p className="mt-6 text-sm leading-6 text-zinc-600 dark:text-zinc-300">
            Public auth flows, protected user flows, and 2FA screens will be added
            on top of this shell in later PRD phases.
          </p>
        </aside>
      </section>
    </main>
  );
}
