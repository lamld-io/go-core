const ACCESS_TOKEN_KEY = "base.auth.access-token";
const REFRESH_TOKEN_KEY = "base.auth.refresh-token";

function getBrowserStorage(): Storage | null {
  if (typeof window === "undefined") {
    return null;
  }

  return window.localStorage;
}

export function getAccessToken(): string | null {
  return getBrowserStorage()?.getItem(ACCESS_TOKEN_KEY) ?? null;
}

export function setAccessToken(token: string): void {
  getBrowserStorage()?.setItem(ACCESS_TOKEN_KEY, token);
}

export function clearAccessToken(): void {
  getBrowserStorage()?.removeItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  return getBrowserStorage()?.getItem(REFRESH_TOKEN_KEY) ?? null;
}

export function setRefreshToken(token: string): void {
  getBrowserStorage()?.setItem(REFRESH_TOKEN_KEY, token);
}

export function clearRefreshToken(): void {
  getBrowserStorage()?.removeItem(REFRESH_TOKEN_KEY);
}

export function clearAuthTokens(): void {
  clearAccessToken();
  clearRefreshToken();
}
