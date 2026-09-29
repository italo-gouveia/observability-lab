"""alertsink — a tiny AlertManager webhook receiver.

Logs every alert it receives so the full alerting path (Prometheus rule ->
AlertManager -> receiver) is visible end to end without a paid integration.
Stands in for PagerDuty / OpsGenie / Slack. Pure stdlib, no dependencies.
"""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        raw = self.rfile.read(length)
        try:
            payload = json.loads(raw)
            for alert in payload.get("alerts", []):
                name = alert.get("labels", {}).get("alertname", "?")
                severity = alert.get("labels", {}).get("severity", "?")
                summary = alert.get("annotations", {}).get("summary", "")
                print(f"[{alert.get('status', '?').upper()}] {name} "
                      f"(severity={severity}) - {summary}", flush=True)
        except Exception as exc:  # noqa: BLE001 - demo sink, log and move on
            print(f"failed to parse alert payload: {exc}", flush=True)
        self.send_response(200)
        self.end_headers()

    def do_GET(self):  # simple health endpoint
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")

    def log_message(self, *args):  # silence default access logging
        pass


if __name__ == "__main__":
    print("alertsink listening on :5001", flush=True)
    HTTPServer(("0.0.0.0", 5001), Handler).serve_forever()
