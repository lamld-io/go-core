import test from "node:test";
import assert from "node:assert/strict";

import { ApiClientError, parseApiResponse } from "@/lib/api/client";
import { parseAppEnv } from "@/lib/env";
import { getAccessToken } from "@/lib/auth/storage";

test("parseAppEnv returns normalized public config", () => {
  const env = parseAppEnv({
    NEXT_PUBLIC_APP_NAME: "Portal",
    NEXT_PUBLIC_AUTH_API_BASE_URL: "http://localhost:8080/",
  } as unknown as NodeJS.ProcessEnv);

  assert.deepEqual(env, {
    appName: "Portal",
    authApiBaseUrl: "http://localhost:8080",
  });
});

test("parseAppEnv rejects missing auth api base url", () => {
  assert.throws(
    () => parseAppEnv({} as unknown as NodeJS.ProcessEnv),
    /NEXT_PUBLIC_AUTH_API_BASE_URL/,
  );
});

test("parseApiResponse returns typed data for successful envelopes", async () => {
  const response = new Response(
    JSON.stringify({
      code: "SUCCESS",
      message: "success",
      data: { id: "123" },
    }),
    {
      status: 200,
      headers: { "Content-Type": "application/json" },
    },
  );

  const data = await parseApiResponse<{ id: string }>(response);
  assert.deepEqual(data, { id: "123" });
});

test("parseApiResponse throws ApiClientError for backend error envelopes", async () => {
  const response = new Response(
    JSON.stringify({
      code: "UNAUTHORIZED",
      message: "invalid credentials",
    }),
    {
      status: 401,
      headers: { "Content-Type": "application/json" },
    },
  );

  await assert.rejects(() => parseApiResponse(response), (error: unknown) => {
    assert.ok(error instanceof ApiClientError);
    assert.equal(error.status, 401);
    assert.equal(error.code, "UNAUTHORIZED");
    return true;
  });
});

test("parseApiResponse safely handles 204 responses", async () => {
  const response = new Response(null, { status: 204 });
  const data = await parseApiResponse(response);
  assert.equal(data, undefined);
});

test("auth storage does not crash during server-side import", () => {
  assert.equal(getAccessToken(), null);
});
