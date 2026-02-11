"""
Minimal Flask app for delivery status. Used by OrderB2bAPI (HTTP).
Loads .env from this directory (GetDeliveryStatus) for Wassel credentials.
"""
import os
from flask import Flask, request, jsonify

# Load .env from GetDeliveryStatus directory
try:
    from dotenv import load_dotenv
    _env_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".env")
    load_dotenv(_env_path)
except ImportError:
    pass

from connect import get_shipment_details

app = Flask(__name__)


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


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 5000))
    app.run(host="0.0.0.0", port=port)
