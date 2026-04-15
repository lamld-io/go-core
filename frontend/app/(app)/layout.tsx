import type { ReactNode } from "react";

export default function ProtectedLayout({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto flex min-h-screen w-full max-w-4xl flex-col px-6 py-12 sm:px-10">
      <div className="mb-8 flex items-center justify-between gap-4">
        <div>
          <p className="text-sm font-medium uppercase tracking-[0.2em] text-zinc-500">
            Protected App Area
          </p>
          <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-300">
            Auth guards and session-aware views will be added in later phases.
          </p>
        </div>
      </div>
      <div className="rounded-3xl border border-black/10 bg-white p-8 shadow-sm dark:border-white/10 dark:bg-white/5">
        {children}
      </div>
    </div>
  );
}
