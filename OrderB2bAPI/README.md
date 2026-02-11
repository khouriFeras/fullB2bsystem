# JafarShop B2B API

A Go-based REST API service that enables partner stores to submit orders containing JafarShop products. The system detects supplier SKUs, creates Shopify Draft Orders, supports manual confirmation, and provides order status polling.

## Features

- **Partner Authentication**: API key-based authentication for partner stores
- **Order Intake**: Receive full cart submissions from partners
- **SKU Detection**: Automatically detects if cart contains supplier products
- **Shopify Integration**: Creates draft orders with variants (supplier items) and custom items (non-supplier items)
- **Manual Confirmation**: Admin endpoints for confirming/rejecting orders
- **Order Status**: Polling endpoint for partners to check order status
- **Shipping Tracking**: Update and track shipping information
- **Idempotency**: Prevents duplicate orders with idempotency keys
- **Audit Trail**: Complete event logging for order lifecycle

## Architecture

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **Database**: PostgreSQL 14+
- **Shopify API**: GraphQL Admin API

## Project Structure

```
B2BAPI/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and middleware
│   ├── domain/         # Domain models and enums
│   ├── repository/     # Data access layer
│   ├── service/        # Business logic
│   ├── shopify/        # Shopify API client
│   └── config/         # Configuration management
├── migrations/         # Database migrations
└── pkg/errors/         # Custom error types
```

## Setup

### Prerequisites

- Go 1.21 or later
- PostgreSQL 14 or later
- Docker and Docker Compose (for local development)
- Shopify store with Admin API access

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd B2BAPI
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up PostgreSQL**
   ```bash
   docker-compose up -d
   ```

4. **Run migrations**
   ```bash
   # Install golang-migrate if not already installed
   # macOS: brew install golang-migrate
   # Linux: https://github.com/golang-migrate/migrate/releases
   
   migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/b2bapi?sslmode=disable" up
   ```

5. **Configure environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

6. **Set up Shopify App**
   - Follow the guide in `scripts/setup_shopify_app.md`
   - Add `SHOPIFY_SHOP_DOMAIN` and `SHOPIFY_ACCESS_TOKEN` to `.env`

7. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

## Configuration

Environment variables (see `env.example`):

- `PORT` - Server port (default: 8080)
- `ENVIRONMENT` - Environment (development/production)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - Database configuration
- `SHOPIFY_SHOP_DOMAIN` - Your Shopify store domain
- `SHOPIFY_ACCESS_TOKEN` - Shopify Admin API access token
- `API_KEY_HASH_SALT` - Salt for API key hashing
- `LOG_LEVEL` - Logging level (debug/info/warn/error)

## API Endpoints

### Partner Endpoints

#### POST /v1/carts/submit
Submit a cart for processing.

**Headers:**
- `Authorization: Bearer {api_key}`
- `Idempotency-Key: {uuid}` (optional)

**Request Body:**
```json
{
  "partner_order_id": "order-123",
  "items": [
    {
      "sku": "PROD-001",
      "title": "Product Name",
      "price": 29.99,
      "quantity": 2,
      "product_url": "https://example.com/product"
    }
  ],
  "customer": {
    "name": "John Doe",
    "phone": "+1234567890"
  },
  "shipping": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "postal_code": "10001",
    "country": "US"
  },
  "totals": {
    "subtotal": 59.98,
    "tax": 4.80,
    "shipping": 5.00,
    "total": 69.78
  },
  "payment_status": "paid"
}
```

**Response:**
- `200 OK`: Order created
- `204 No Content`: No supplier SKUs in cart
- `409 Conflict`: Idempotency key conflict
- `422 Unprocessable Entity`: Validation error

#### GET /v1/orders/{id}
Get order status.

**Headers:**
- `Authorization: Bearer {api_key}`

**Response:**
```json
{
  "id": "uuid",
  "partner_order_id": "order-123",
  "status": "PENDING_CONFIRMATION",
  "shopify_draft_order_id": 123456,
  "customer_name": "John Doe",
  "shipping_address": {...},
  "cart_total": 69.78,
  "items": [...],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Admin Endpoints

#### POST /v1/admin/orders/{id}/confirm
Confirm an order.

#### POST /v1/admin/orders/{id}/reject
Reject an order.

**Request Body:**
```json
{
  "reason": "Out of stock"
}
```

#### POST /v1/admin/orders/{id}/ship
Mark order as shipped with tracking.

**Request Body:**
```json
{
  "carrier": "Standard Shipping",
  "tracking_number": "TRACK123456789",
  "tracking_url": "https://example.com/track/TRACK123456789"
}
```

#### GET /v1/admin/orders
List orders (with query parameters: `status`, `limit`, `offset`).

## Order Status Flow

```
PENDING_CONFIRMATION → CONFIRMED → SHIPPED → DELIVERED
                    ↓
                 REJECTED
                    ↓
                 CANCELLED (from any state)
```

## SKU Mapping

The system maintains a mapping of SKUs to Shopify variants. To sync SKUs from Shopify:

1. Query Shopify products using the GraphQL API
2. Store SKU mappings in the `sku_mappings` table
3. Only orders with at least one mapped SKU are processed

## Partner Setup

1. Create a partner record in the database
2. Generate an API key (hash it with bcrypt)
3. Store the hash in the `partners` table
4. Provide the API key to the partner

Example SQL:
```sql
INSERT INTO partners (name, api_key_hash, is_active)
VALUES ('Zain Shop', '<bcrypt_hash_of_api_key>', true);
```

## Development

### Running Tests
```bash
go test ./...
```

### Database Migrations
```bash
# Up
migrate -path ./migrations -database "postgres://..." up

# Down
migrate -path ./migrations -database "postgres://..." down
```

## Production Considerations

- Use environment-specific configuration
- Set up proper logging and monitoring
- Implement rate limiting
- Use HTTPS only
- Rotate API keys periodically
- Set up database backups
- Consider adding a lookup hash column for API keys (SHA256) for efficient authentication
- Implement webhook delivery system (currently polling only)

## License

[Your License Here]
