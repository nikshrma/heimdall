import http from "k6/http";
import { check } from "k6";
import { Counter, Rate } from "k6/metrics";

const GATEWAY_URL = "http://localhost:8080";

const rateLimited = new Counter("rate_limited");
const serverErrors = new Rate("server_errors");

const backend1Count = new Counter("backend1_hits");
const backend2Count = new Counter("backend2_hits");
const backend3Count = new Counter("backend3_hits");

export const options = {
  scenarios: {
    load: {
      executor: "constant-arrival-rate",
      rate: 6000,
      timeUnit: "1s",
      duration: "60s",
      preAllocatedVUs: 7000,
      maxVUs: 10000,
    },
  },
};

export default function () {
  const res = http.post(`${GATEWAY_URL}/users`, null, {
    timeout: "3s",
  });

  if (res.status === 429) {
    rateLimited.add(1);
  }

  if (res.status >= 500) {
    serverErrors.add(1);
  }

  const ok = check(res, {
    "request succeeded": (r) => r.status === 200,
  });

  if (ok) {
    try {
      const body = JSON.parse(res.body);

      if (body.name === "backend1") backend1Count.add(1);
      if (body.name === "backend2") backend2Count.add(1);
      if (body.name === "backend3") backend3Count.add(1);
    } catch (e) {}
  }
}
