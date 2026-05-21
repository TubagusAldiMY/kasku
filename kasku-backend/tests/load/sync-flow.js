// k6 load test — Sync flow: login → push 10 ops → pull
// Run: k6 run --env BASE_URL=http://localhost:8080 \
//            --env TEST_USER_EMAIL=verified@kasku.test \
//            --env TEST_USER_PASSWORD=TestPass1! \
//            sync-flow.js
import http from "k6/http";
import { check, sleep } from "k6";
import { smokeOptions } from "./k6-config.js";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TEST_EMAIL = __ENV.TEST_USER_EMAIL || "";
const TEST_PASS = __ENV.TEST_USER_PASSWORD || "";

export const options = {
  ...smokeOptions,
  vus: 20,
  duration: "2m",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1000"],  // Sync push: P95 < 1s
  },
};

export function setup() {
  if (!TEST_EMAIL || !TEST_PASS) {
    throw new Error("Set TEST_USER_EMAIL dan TEST_USER_PASSWORD");
  }
  const res = http.post(
    `${BASE_URL}/v1/auth/login`,
    JSON.stringify({ email: TEST_EMAIL, password: TEST_PASS }),
    { headers: { "Content-Type": "application/json" } }
  );
  if (res.status !== 200) {
    throw new Error(`Login gagal: ${res.status}`);
  }
  return { accessToken: res.json("data.access_token") };
}

function buildPushPayload(count) {
  const ops = [];
  for (let i = 0; i < count; i++) {
    ops.push({
      sync_id: uuidv4(),
      entity_type: "transaction",
      operation: "INSERT",
      entity_id: uuidv4(),
      client_timestamp: new Date().toISOString(),
      payload: {
        account_id: uuidv4(),
        amount: 5000 + i * 100,
        type: "EXPENSE",
        description: `k6 sync test op ${i}`,
        date: new Date().toISOString().split("T")[0],
      },
    });
  }
  return { operations: ops };
}

export default function (data) {
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${data.accessToken}`,
  };

  // 1. Push 10 operasi
  const pushRes = http.post(
    `${BASE_URL}/v1/sync/push`,
    JSON.stringify(buildPushPayload(10)),
    { headers }
  );
  check(pushRes, {
    "sync push: status 200": (r) => r.status === 200,
    "sync push: no server error": (r) => r.status < 500,
  });

  sleep(0.5);

  // 2. Pull delta
  const since = new Date(Date.now() - 60_000).toISOString();
  const pullRes = http.get(
    `${BASE_URL}/v1/sync/pull?since=${encodeURIComponent(since)}`,
    { headers }
  );
  check(pullRes, {
    "sync pull: status 200": (r) => r.status === 200,
  });

  sleep(1);
}
