"""
Minimal Flask app for delivery status. Used by OrderB2bAPI (HTTP).
Loads .env from this directory (GetDeliveryStatus) for Wassel credentials.
- GET /shipment: outbound call to Wassel API (pull).
- POST /webhooks/wassel/status: inbound webhook from Wassel (push); Bearer auth.
  On accept, forwards to OrderB2bAPI /internal/webhooks/delivery so the partner can be notified.
"""
import os
import logging
from flask import Flask, request, jsonify

import requests

# Load .env from GetDeliveryStatus directory
try:
    from dotenv import load_dotenv
    _env_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env")
    load_dotenv(_env_path)
except ImportError:
    pass

from connect import get_shipment_details

app = Flask(__name__)

# --- Wassel inbound webhook helpers ---

def _forward_to_order_b2b_api(payload):
    """
    POST the delivery payload to OrderB2bAPI internal webhook so it can notify the partner.
    Only forwards if ORDER_B2B_API_URL and DELIVERY_WEBHOOK_SECRET are set.
    Always returns None; logs on failure. Caller should still return 200 to Wassel.
    """
    base_url = os.environ.get("ORDER_B2B_API_URL") or os.environ.get("ORDER_B2B_API_INTERNAL_URL")
    secret = os.environ.get("DELIVERY_WEBHOOK_SECRET")
    if not (base_url and str(base_url).strip() and secret and str(secret).strip()):
        return
    url = f"{base_url.rstrip('/')}/internal/webhooks/delivery"
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {secret}"}
    try:
        r = requests.post(url, json=payload, headers=headers, timeout=5)
        if r.status_code >= 400:
            logging.warning("OrderB2bAPI delivery webhook returned %s: %s", r.status_code, r.text[:500])
    except Exception as e:
        logging.warning("OrderB2bAPI delivery webhook request failed: %s", e)


def _get_bearer_token():
    """Extract Bearer token from Authorization header. Returns None if missing or invalid format."""
    auth = request.headers.get("Authorization")
    if not auth or not isinstance(auth, str):
        return None
    auth = auth.strip()
    if auth.lower().startswith("bearer "):
        return auth[7:].strip()
    return None


def _validate_wassel_format(payload):
    """
    Validate Wassel's native format: ItemReferenceNo (string), Status (integer).
    Accepts ItemReferenceNo or itemReferenceNo, Status or status (casing may vary).
    Returns None if valid, else error message string.
    """
    if not isinstance(payload, dict):
        return "Body must be a JSON object"
    ref = payload.get("ItemReferenceNo") or payload.get("itemReferenceNo")
    if ref is None:
        return "ItemReferenceNo is required"
    if not str(ref).strip():
        return "ItemReferenceNo must be non-empty"
    status = payload.get("Status") or payload.get("status")
    if status is None:
        return "Status is required"
    try:
        int(status)
    except (TypeError, ValueError):
        return "Status must be an integer"
    return None


def _validate_single_event(data, waybill_top=None, order_ref_top=None):
    """
    Validate a single event (from top-level payload or from events[]).
    Required: event_id, waybill (or waybill_top), status.code, occurred_at.
    Returns (None, None) if valid, else (error_message, 400).
    """
    event_id = data.get("event_id") if isinstance(data.get("event_id"), str) else None
    waybill = (data.get("waybill") or waybill_top) if isinstance(data.get("waybill"), (str, type(None))) else None
    if waybill_top is not None:
        waybill = waybill or waybill_top
    status = data.get("status")
    code = status.get("code") if isinstance(status, dict) and status else None
    if isinstance(code, str):
        code = code.strip()
    else:
        code = None
    occurred_at = data.get("occurred_at") if isinstance(data.get("occurred_at"), str) else None

    if not (event_id and event_id.strip()):
        return "event_id is required and must be a non-empty string", 400
    if not (waybill and str(waybill).strip()):
        return "waybill is required (per event or top-level for bulk)", 400
    if not (code and code.strip()):
        return "status.code is required and must be a non-empty string", 400
    if not (occurred_at and occurred_at.strip()):
        return "occurred_at is required (ISO-8601 datetime string)", 400
    return None, None


def _normalize_events(payload):
    """
    Parse payload as either single event or bulk (events array).
    Returns (events_list, waybill, order_ref) or (None, None, None) with error.
    Each event is a dict with at least event_id, waybill, status.code, occurred_at.
    """
    if not isinstance(payload, dict):
        return None, None, None, "Body must be a JSON object"
    waybill_top = payload.get("waybill")
    if isinstance(waybill_top, str):
        waybill_top = waybill_top.strip() or None
    order_ref_top = payload.get("order_ref")
    if isinstance(order_ref_top, str):
        order_ref_top = order_ref_top.strip() or None

    events_raw = payload.get("events")
    if events_raw is not None:
        if not isinstance(events_raw, list):
            return None, None, None, "events must be an array"
        if not (waybill_top and str(waybill_top).strip()):
            return None, None, None, "waybill is required at top-level for bulk events"
        events = []
        for i, ev in enumerate(events_raw):
            if not isinstance(ev, dict):
                return None, None, None, f"events[{i}] must be an object"
            err, _ = _validate_single_event(ev, waybill_top=waybill_top, order_ref_top=order_ref_top)
            if err:
                return None, None, None, err
            # Ensure waybill on each event for consistency
            ev_waybill = ev.get("waybill") or waybill_top
            events.append({**ev, "waybill": ev_waybill, "order_ref": ev.get("order_ref") or order_ref_top})
        return events, waybill_top, order_ref_top, None
    # Single event
    err, _ = _validate_single_event(payload, waybill_top=waybill_top, order_ref_top=order_ref_top)
    if err:
        return None, None, None, err
    waybill = payload.get("waybill") or waybill_top
    return [payload], waybill, payload.get("order_ref") or order_ref_top, None


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"}), 200


@app.route("/shipment", methods=["GET"])
def shipment():
    """
    GET /shipment?reference_id=...&awb=...&partner_id=...
    At least one of reference_id or awb is required. partner_id is optional (for logging / scoping).
    Returns Wassel API JSON (or error dict with status_code).
    """
    reference_id = request.args.get("reference_id")
    awb = request.args.get("awb")
    partner_id = request.args.get("partner_id")

    if reference_id is None and awb is None:
        return jsonify({"error": "Provide at least one of reference_id or awb"}), 400

    # Normalize: empty string -> None for optional
    if reference_id == "":
        reference_id = None
    if awb == "":
        awb = None
    if partner_id == "":
        partner_id = None
    if reference_id is None and awb is None:
        return jsonify({"error": "Provide at least one of reference_id or awb"}), 400

    try:
        result = get_shipment_details(reference_id=reference_id, awb=awb, partner_id=partner_id)
        return jsonify(result), 200
    except ValueError as e:
        return jsonify({"error": str(e)}), 400
    except Exception as e:
        return jsonify({"error": str(e)}), 500


@app.route("/webhooks/wassel/status", methods=["POST"])
def webhook_wassel_status():
    """
    Inbound webhook from Wassel. We accept Wassel's native format (ItemReferenceNo + Status int)
    or the extended format (event_id, waybill, status.code, occurred_at, etc.).
    Headers: Content-Type: application/json, Authorization: Bearer <WASSEL_SHARED_SECRET>
    Success: 200 OK with { "ok": true }
    """
    secret = os.environ.get("WASSEL_SHARED_SECRET")
    if not (secret and str(secret).strip()):
        return jsonify({"error": "webhook not configured"}), 503
    token = _get_bearer_token()
    if not token or token != secret:
        return jsonify({"error": "unauthorized"}), 401
    if request.content_type and "application/json" not in request.content_type:
        return jsonify({"error": "Content-Type must be application/json"}), 400
    try:
        payload = request.get_json(force=True, silent=False)
    except Exception:
        return jsonify({"error": "invalid JSON body"}), 400
    # Prefer Wassel's native format (ItemReferenceNo + Status)
    err = _validate_wassel_format(payload)
    if err is None:
        _forward_to_order_b2b_api(payload)
        return jsonify({"ok": True}), 200
    # Fallback: extended format (event_id, waybill, status.code, occurred_at)
    events, _waybill, _order_ref, err_ext = _normalize_events(payload)
    if err_ext:
        return jsonify({"error": err or err_ext}), 400
    return jsonify({"ok": True}), 200


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5000))
    app.run(host="0.0.0.0", port=port)
