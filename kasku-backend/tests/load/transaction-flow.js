// k6 load test — Transaction flow: login → list accounts → create transaction → get balance
// Membutuhkan user yang sudah verified dan tenant yang sudah di-provision.
// Siapkan via: TEST_USER_EMAIL dan TEST_USER_PASSWORD di env.
// Run: k6 run --env BASE_URL=http://localhost:8080 \
//            --env TEST_USER_EMAIL=verified@kasku.test \
//            --env TEST_USER_PASSWORD=TestPass1! \
//            transaction-flow.js
import http from "k6/http";
import { check, sleep } from "k6";
import { stageOptions } from "./k6-config.js";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TEST_EMAIL = __ENV.TEST_USER_EMAIL || "";
const TEST_PASS = __ENV.TEST_USER_PASSWORD || "";

export const options = {
  ...stageOptions,
  thresholds: {
    ...stageOptions.thresholds,
    http_req_duration: ["p(95)<300"],  // Transaction: P95 < 300ms
  },
};

// Setup: login sekali dan simpan token (dipakai semua VU)
export function setup() {
  if (!TEST_EMAIL || !TEST_PASS) {
    throw new Error(
      "Set TEST_USER_EMAIL dan TEST_USER_PASSWORD — butuh user verified dengan tenant aktif"
    );
  }
  const res = http.post(
    `${BASE_URL}/v1/auth/login`,
    JSON.stringify({ email: TEST_EMAIL, password: TEST_PASS }),
    { headers: { "Content-Type": "application/json" } }
  );
  if (res.status !== 200) {
    throw new Error(`Login gagal: ${res.status} — ${res.body}`);
  }
  return { accessToken: res.json("data.access_token") };
}

export default function (data) {
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${data.accessToken}`,
  };

  // 1. List accounts
  const accountsRes = http.get(`${BASE_URL}/v1/accounts`, { headers });
  check(accountsRes, {
    "list accounts: status 200": (r) => r.status === 200,
  });

  const accounts = accountsRes.json("data") || [];
  if (accounts.length === 0) {
    sleep(1);
    return;
  }

  const accountId = accounts[0].id;
  sleep(0.2);

  // 2. Create transaction
  const txRes = http.post(
    `${BASE_URL}/v1/transactions`,
    JSON.stringify({
      account_id: accountId,
      amount: 10000,
      type: "EXPENSE",
      description: "Load test transaction",
      date: new Date().toISOString().split("T")[0],
    }),
    { headers }
  );
  check(txRes, {
    "create transaction: status 201": (r) => r.status === 201,
  });

  sleep(0.2);

  // 3. Get account detail (lihat balance)
  const detailRes = http.get(`${BASE_URL}/v1/accounts/${accountId}`, { headers });
  check(detailRes, {
    "get account detail: status 200": (r) => r.status === 200,
  });

  sleep(1);
}
