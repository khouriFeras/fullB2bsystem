import json
import os
import requests

try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass

# Which store to use: "test" or "actual" (from .env STORE_MODE, default "test")
STORE_MODE = (os.environ.get("STORE_MODE") or "test").strip().lower()
if STORE_MODE not in ("test", "actual"):
    STORE_MODE = "test"
PREFIX = "TEST_" if STORE_MODE == "test" else "ACTUAL_"


def get_store_credentials():
    """Return Email, Password, CompanyId, StoreId, Token, Base_url for current STORE_MODE."""
    return {
        "email": os.environ.get(f"{PREFIX}Email"),
        "password": os.environ.get(f"{PREFIX}Password"),
        "company_id": os.environ.get(f"{PREFIX}CompanyId"),
        "store_id": os.environ.get(f"{PREFIX}StoreId"),
        "token": os.environ.get(f"{PREFIX}Token"),
        "base_url": os.environ.get(f"{PREFIX}Base_url"),
    }


DELIVERY_STATUS_PATHS_JSON = [
    "Company/GetShipmentItemActionsWithMatchingItem",
    "Integration/TrackShipment",
    "Integration/GetShipmentStatus",
    "Integration/GetShipmentByReference",
]


def get_shipment_details(reference_id=None, awb=None, partner_id=None, token=None, base_url=None, try_alternate_paths=True):
    """Get shipment details by reference_id and/or awb. Optional partner_id for logging/future per-partner credentials. Returns API JSON."""
    creds = get_store_credentials()
    token = token or creds["token"]
    api_base = base_url or creds["base_url"]
    if not token:
        raise ValueError(f"No token: set {PREFIX}Token in .env or pass token=...")
    if not api_base:
        raise ValueError(f"No Base_url: set {PREFIX}Base_url in .env")
    headers = {"Authorization": f"Bearer {token}"}
    params = {}
    # When reference_id is provided, use only referenceID so Wassel returns that order (not another order matched by awb)
    if reference_id is not None and (not isinstance(reference_id, str) or reference_id.strip() != ""):
        params["referenceID"] = reference_id
    elif awb is not None and (not isinstance(awb, str) or awb.strip() != ""):
        params["awb"] = awb
    else:
        raise ValueError("Provide reference_id (order number) or awb")

    base = api_base.rstrip("/")
    verbose = os.environ.get("VERBOSE", "").strip().lower() in ("1", "true", "yes")
    if verbose and partner_id:
        print(f"Partner ID: {partner_id}")
    last_response = None
    # When looking up by reference_id, try GetShipmentByReference first (reference-specific endpoint)
    paths = list(DELIVERY_STATUS_PATHS_JSON)
    if "referenceID" in params and "Integration/GetShipmentByReference" in paths:
        paths.remove("Integration/GetShipmentByReference")
        paths.insert(0, "Integration/GetShipmentByReference")

    for path in paths:
        url = f"{base}/{path}"
        # Wassel demo uses awb (not referenceID) for GetShipmentItemActionsWithMatchingItem
        req_params = dict(params)
        if path == "Company/GetShipmentItemActionsWithMatchingItem" and "referenceID" in req_params:
            req_params["awb"] = req_params.pop("referenceID", None) or reference_id
        full_url = requests.Request("GET", url, params=req_params).prepare().url
        if verbose:
            print(f"Request: GET {full_url}")
        response = requests.get(url, params=req_params, headers=headers, timeout=15)
        last_response = response

        if response.status_code != 200:
            if response.status_code == 404 and try_alternate_paths and verbose:
                print("  -> 404, trying next path...")
            continue

        ct = response.headers.get("Content-Type", "") or ""
        if "application/json" not in ct and response.text:
            try:
                response.json()
            except Exception:
                continue

        try:
            json_result = response.json()
            if isinstance(json_result, dict):
                json_result.setdefault("httpStatusCode", response.status_code)
                if partner_id:
                    json_result["partner_id"] = partner_id
            return _filter_response_by_reference(json_result, reference_id)
        except Exception:
            continue

    try:
        out = last_response.json()
        if isinstance(out, dict) and partner_id:
            out["partner_id"] = partner_id
        return _filter_response_by_reference(out, reference_id)
    except Exception:
        result = {"status_code": last_response.status_code, "text": (last_response.text[:500] if last_response.text else "(empty)")}
        if last_response.status_code == 404:
            result["_note"] = "404 – check endpoint with Wassel API docs."
        if partner_id:
            result["partner_id"] = partner_id
        return result


def _normalize_reference(reference_id):
    """Return reference_id as string for comparison, or None if empty."""
    if reference_id is None:
        return None
    s = str(reference_id).strip()
    return s if s else None


def _get_shipment_no_from_response(json_result):
    """Extract shipment number from response if present; return None if not found."""
    try:
        data = json_result.get("data") or {}
        result_data = data.get("result", {}).get("data") or {}
        if result_data.get("shipmentNo") is not None:
            return str(result_data.get("shipmentNo")).strip()
        matches = data.get("matchs", {}).get("data") or []
        for m in matches:
            sn = m.get("shipmentNo")
            if sn is not None:
                return str(sn).strip()
        return None
    except Exception:
        return None


def _filter_response_by_reference(json_result, reference_id):
    """Return only the shipment that matches reference_id; 404 only when response clearly has a different order."""
    ref_str = _normalize_reference(reference_id)
    if ref_str is None:
        return json_result
    returned_shipment_no = _get_shipment_no_from_response(json_result)
    # If we got a 200 and the response has a shipment number, it must match what we asked for
    if returned_shipment_no is not None and returned_shipment_no != ref_str:
        return {
            "status_code": 404,
            "error": f"No shipment found for order number {ref_str}",
            "_note": "Response did not match the requested order number.",
        }
    # No shipmentNo in response (e.g. different endpoint structure) or it matches – return as-is
    # Filter data.matchs.data to only the matching shipment(s)
    try:
        data = json_result.get("data")
        if data and isinstance(data, dict):
            matchs = data.get("matchs", {}).get("data")
            if isinstance(matchs, list) and len(matchs) > 1:
                filtered = [m for m in matchs if str(m.get("shipmentNo") or "").strip() == ref_str]
                if filtered:
                    data.setdefault("matchs", {})["data"] = filtered
    except Exception:
        pass
    return json_result


if __name__ == "__main__":
    creds = get_store_credentials()
    print(f"Using store: {STORE_MODE} (Base_url: {creds['base_url']})")
    print("Getting shipment details...")
    # Use reference_id only so Wassel returns this order (1034); awb is ignored when reference_id is set
    result = get_shipment_details(reference_id="1034", token=creds["token"], base_url=creds["base_url"])
    print(json.dumps(result, indent=2, ensure_ascii=False))
    if isinstance(result, dict) and result.get("status_code") == 404 and result.get("_note"):
        print("\n" + result["_note"])
