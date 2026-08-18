import http from "k6/http";
import { check } from "k6";
import { Counter, Rate } from "k6/metrics";

const GATEWAY_URL = "http://localhost:8080";

const backend4Count = new Counter("backend4_hits");
const backend5Count = new Counter("backend5_hits");
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
  const res = http.get(`${GATEWAY_URL}/contacts`, {
    timeout: "3s",
  });

  const ok = check(res, {
    "status is 200": (r) => r.status === 200,
  });

  errorRate.add(!ok);

  if (ok) {
    try {
      const body = JSON.parse(res.body);

      if (body.name === "backend4") {
        backend4Count.add(1);
      }

      if (body.name === "backend5") {
        backend5Count.add(1);
      }
    } catch (e) {}
  }
  if (!ok) {
    console.log(`failed: status=${res.status}, error=${res.error}`);
  }
}
