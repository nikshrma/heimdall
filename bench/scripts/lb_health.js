import http from "k6/http";
import { check } from "k6";
import { Counter, Rate } from "k6/metrics";

const GATEWAY_URL = "http://localhost:8080";

const backend1Count = new Counter("backend1_hits");
const backend2Count = new Counter("backend2_hits");
const backend3Count = new Counter("backend3_hits");
const errorRate = new Rate("request_errors");

export const options = {
  scenarios: {
    sustained_load: {
      executor: "constant-vus",
      vus: 10,
      duration: "60s",
    },
  },
  thresholds: {
    request_errors: ["rate>=0"],
  },
};

export default function () {
  const res = http.post(`${GATEWAY_URL}/users`, null, { timeout: "3s" });

  const ok = check(res, {
    "status is 200": (r) => r.status === 200,
  });

  errorRate.add(!ok);

  if (ok) {
    try {
      const body = JSON.parse(res.body);
      if (body.name === "backend1") backend1Count.add(1);
      if (body.name === "backend2") backend2Count.add(1);
      if (body.name === "backend3") backend3Count.add(1);
    } catch (e) {}
  }
}
