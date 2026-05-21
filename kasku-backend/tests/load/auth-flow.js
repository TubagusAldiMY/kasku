// k6 load test — Auth flow: register → login → refresh → logout
// Run: k6 run --env BASE_URL=http://localhost:8080 auth-flow.js
import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Trend } from "k6/metrics";
import { stageOptions } from "./k6-config.js";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  ...stageOptions,
  thresholds: {
    ...stageOptions.thresholds,
    http_req_duration: ["p(95)<500"],  // Auth: P95 < 500ms
  },
};

const loginErrors = new Counter("login_errors");
const loginDuration = new Trend("login_duration_ms");

function randomEmail() {
  return `loadtest_${Math.random().toString(36).slice(2, 10)}@kasku-test.invalid`;
}

export default function () {
  const email = randomEmail();
  const password = "LoadTest1!";
  const username = `lt_${Math.random().toString(36).slice(2, 8)}`;
  const headers = { "Content-Type": "application/json" };

  // 1. Register
  const registerRes = http.post(
    `${BASE_URL}/v1/auth/register`,
    JSON.stringify({ email, password, username }),
    { headers }
  );
  check(registerRes, {
    "register: status 201": (r) => r.status === 201,
  });

  sleep(0.5);

  // 2. Login (email mungkin belum verified, tapi endpoint tetap perlu ditest)
  const start = Date.now();
  const loginRes = http.post(
    `${BASE_URL}/v1/auth/login`,
    JSON.stringify({ email, password }),
    { headers }
  );
  loginDuration.add(Date.now() - start);

  const loginOk = check(loginRes, {
    "login: status 200 atau 403 (unverified)": (r) =>
      r.status === 200 || r.status === 403,
  });

  if (!loginOk) {
    loginErrors.add(1);
    return;
  }

  if (loginRes.status !== 200) {
    sleep(1);
    return;
  }

  const body = loginRes.json();
  const accessToken = body?.data?.access_token;
  if (!accessToken) {
    loginErrors.add(1);
    return;
  }

  const authHeaders = {
    ...headers,
    Authorization: `Bearer ${accessToken}`,
  };

  sleep(0.3);

  // 3. Refresh token
  const refreshRes = http.post(
    `${BASE_URL}/v1/auth/refresh`,
    null,
    { headers: authHeaders }
  );
  check(refreshRes, {
    "refresh: status 200": (r) => r.status === 200,
  });

  sleep(0.3);

  // 4. Logout
  const logoutRes = http.post(
    `${BASE_URL}/v1/auth/logout`,
    null,
    { headers: authHeaders }
  );
  check(logoutRes, {
    "logout: status 200": (r) => r.status === 200,
  });

  sleep(1);
}
