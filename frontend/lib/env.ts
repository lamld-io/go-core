export const DEFAULT_APP_NAME = "Base Auth Portal";

export type AppEnv = {
  appName: string;
  authApiBaseUrl: string;
};

export function parseAppEnv(source: NodeJS.ProcessEnv): AppEnv {
  const authApiBaseUrl = source.NEXT_PUBLIC_AUTH_API_BASE_URL?.trim();

  if (!authApiBaseUrl) {
    throw new Error(
      "NEXT_PUBLIC_AUTH_API_BASE_URL is required to connect the frontend to the auth service.",
    );
  }

  return {
    appName: source.NEXT_PUBLIC_APP_NAME?.trim() || DEFAULT_APP_NAME,
    authApiBaseUrl: authApiBaseUrl.replace(/\/$/, ""),
  };
}

export function getAppEnv(): AppEnv {
  return parseAppEnv(process.env);
}

export function tryGetAppEnv():
  | { ok: true; env: AppEnv }
  | { ok: false; message: string } {
  try {
    return { ok: true, env: getAppEnv() };
  } catch (error) {
    return {
      ok: false,
      message: error instanceof Error ? error.message : "Invalid frontend environment.",
    };
  }
}
