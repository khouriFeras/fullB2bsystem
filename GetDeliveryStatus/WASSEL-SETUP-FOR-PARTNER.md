# Wassel → JafarShop webhook setup

**Webhook URL**
```
https://webhooks.jafarshop.com/webhooks/wassel/status
```

**Method:** `POST`

**Headers (required)**
```
Content-Type: application/json
Authorization: Bearer b3240d81d89ed60342f9e86e48919fbc83df8ff0811f6f755a23b5cd2247e7ee
```

**Payload:** See attached `WASSEL-WEBHOOK-SPEC.json` for:
- Required fields: `ItemReferenceNo`, `Status` (integer)
- Optional: `Waybill`, `DeliveryImageUrl`
- Status codes we accept

**Minimal example**
```json
{"ItemReferenceNo":"1039","Status":170}
```

**Success:** We respond `200` with `{"ok":true}`.

Please use the **Bearer token** above in every webhook request. Keep this token confidential.
