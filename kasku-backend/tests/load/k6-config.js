// Konfigurasi k6 standar untuk load testing KasKu
// Import dan gunakan di setiap scenario file

export const stageOptions = {
  stages: [
    { duration: "30s", target: 50 },  // ramp-up
    { duration: "2m",  target: 50 },  // steady
    { duration: "30s", target: 0  },  // ramp-down
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],   // error rate < 1%
    http_req_duration: ["p(95)<2000"], // P95 < 2s (fallback umum)
  },
};

// Opsi ringan untuk CI / smoke load test
export const smokeOptions = {
  vus: 5,
  duration: "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<2000"],
  },
};
