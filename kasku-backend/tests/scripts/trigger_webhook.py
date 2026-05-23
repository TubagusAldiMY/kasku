#!/usr/bin/env python3
"""
Trigger billing-service webhook secara manual untuk local dev sandbox testing.
Dipakai ketika orchestrator (remote) tidak bisa menjangkau localhost.

Usage:
    python3 trigger_webhook.py <order_id> [scenario]

    scenario: success (default) | failed | expired

Example:
    python3 trigger_webhook.py KASKU-SUB-xxx-1234567890
    python3 trigger_webhook.py KASKU-SUB-xxx-1234567890 failed
"""
import hmac
import hashlib
import json
import sys
import urllib.request
import urllib.error
import subprocess
import os

BILLING_URL = os.getenv("BILLING_URL", "http://172.19.0.2:8083/v1/billing/webhook/payment")
WEBHOOK_SECRET = os.getenv("PAYMENT_WEBHOOK_SECRET", "CHANGE_THIS_FROM_PORTAL_WEBHOOK_SECRET_REGENERATE")


def get_payment_by_order_id(order_id: str) -> dict | None:
    """Lookup payment record dari DB lewat Docker exec."""
    result = subprocess.run(
        [
            "docker", "exec", "kasku-postgres", "psql",
            "-U", "kasku_superuser", "-d", "kasku_billing", "-tAc",
            f"SELECT order_id, external_payment_id, amount_idr FROM payments "
            f"WHERE order_id='{order_id}' LIMIT 1",
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0 or not result.stdout.strip():
        return None
    parts = result.stdout.strip().split("|")
    return {
        "order_id": parts[0],
        "external_payment_id": parts[1],
        "amount_idr": int(parts[2]),
    }


def trigger(order_id: str, scenario: str = "success") -> None:
    payment = get_payment_by_order_id(order_id)
    if not payment:
        print(f"ERROR: payment dengan order_id '{order_id}' tidak ditemukan di DB")
        sys.exit(1)

    event_map = {
        "success": "payment.success",
        "failed": "payment.failed",
        "expired": "payment.expired",
    }
    event = event_map.get(scenario, "payment.success")

    payload = json.dumps(
        {
            "event": event,
            "paymentId": payment["external_payment_id"],
            "refId": payment["order_id"],
            "amount": payment["amount_idr"],
            "status": scenario,
        },
        separators=(",", ":"),
    ).encode()

    sig = hmac.new(WEBHOOK_SECRET.encode(), payload, hashlib.sha256).hexdigest()

    req = urllib.request.Request(
        BILLING_URL,
        data=payload,
        headers={"Content-Type": "application/json", "X-Signature": sig},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req) as r:
            print(f"OK ({r.status}): {r.read().decode()}")
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        print(f"ERROR ({e.code}): {body}")
        sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    trigger(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else "success")
