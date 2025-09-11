# poptape-lister-redux

Golang microservice for Poptape Auction application list management - recently viewed, watchlists, watchers of items etc.

This is a Golang implementation of the [original Python poptape-lister](https://github.com/cliveyg/poptape-lister) microservice, following the project structure of [poptape-admin](https://github.com/cliveyg/poptape-admin).

Please see [this gist](https://gist.github.com/cliveyg/cf77c295e18156ba74cda46949231d69) to see how this microservice works as part of the auction system software.

## Features

- **MongoDB Integration**: Uses MongoDB for data storage
- **Gin-Gonic Router**: Fast HTTP web framework for Go
- **UUID Support**: Uses `github.com/google/uuid` library
- **Dockerized**: Complete Docker setup with MongoDB
- **Structured Logging**: Uses zerolog for structured logging
- **Middleware Support**: Authentication, CORS, rate limiting, JSON validation

## API Routes

### Authenticated Routes

All authenticated routes require an `X-Public-ID` header containing a valid UUID (in production, this would be extracted from JWT tokens).

#### Watchlist Management
```
GET /list/watchlist
```
Returns a list of item UUIDs for the authenticated user's watchlist.

Example response:
```json
{
    "watchlist": [
        "2a99371f-4188-49b8-a628-85e946540364",
        "803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
    ]
}
```

```
POST /list/watchlist
```
Adds an item to the user's watchlist.

Example request:
```json
{
    "uuid": "2a99371f-4188-49b8-a628-85e946540364"
}
```

```
DELETE /list/watchlist
```
Removes an item from the user's watchlist.

Example request:
```json
{
    "uuid": "2a99371f-4188-49b8-a628-85e946540364"
}
```

#### Recently Viewed Items
```
GET /list/viewed
```
Returns a list of item UUIDs for the authenticated user's recently viewed items.

Example response:
```json
{
    "recently_viewed": [
        "2a99371f-4188-49b8-a628-85e946540364",
        "803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
    ]
}
```

```
POST /list/viewed
```
Adds an item to the user's recently viewed list.

#### Favourite Sellers
```
GET /list/favourites
```
Returns a list of the user's favourite sellers.

Example response:
```json
{   
    "favourites": [
        {
            "username": "user_2a99371f",
            "public_id": "2a99371f-4188-49b8-a628-85e946540364"
        },
        {
            "username": "user_803be8ad", 
            "public_id": "803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
        }
    ]
}
```

```
POST /list/favourites
DELETE /list/favourites
```
Add or remove favourite sellers.

#### Recent Bids
```
GET /list/recentbids
```
Returns the user's recent bids.

Example response:
```json
{
    "recent_bids": [
        {
            "auction_id": "a47cdbb5-2e45-4aef-af71-82736351f049",
            "lot_id": "2a99371f-4188-49b8-a628-85e946540364",
            "amount": 176.99
        }
    ]
}
```

```
POST /list/recentbids
```
Add a recent bid record.

#### Purchase History
```
GET /list/purchased
```
Returns the user's purchase history.

Example response:
```json
{
    "purchased": [
        {
            "purchase_id": "a933d845-bf82-421c-bf5c-57f81c182912",
            "auction_id": "a47cdbb5-2e45-4aef-af71-82736351f049",
            "lot_id": "2a99371f-4188-49b8-a628-85e946540364",
            "amount": 176.99
        }
    ]
}
```

```
POST /list/purchased
```
Add a purchase record.

### Public Routes

#### Watching Count
```
GET /list/watching/<item_id>
```
Returns the total number of people watching an item (unauthenticated).

Example response:
```json
{
    "people_watching": 10
}
```

#### System Status
```
GET /list/status
```
Returns system status (unauthenticated).

Example response:
```json
{
    "message": "System running...",
    "version": "v0.1.0"
}
```

## Project Structure

```
├── lister.go          # Main entry point
├── app.go             # App initialization
├── models.go          # Data models and structures
├── handlers.go        # HTTP request handlers
├── routes.go          # Route definitions
├── database.go        # MongoDB connection and operations
├── middleware.go      # Authentication and other middleware
├── helpers.go         # Helper functions
├── utils/
│   └── utils.go       # Utility functions
├── Dockerfile         # Docker configuration
├── docker-compose.yml # Docker Compose setup
├── .env.example       # Environment configuration template
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
└── README.md         # This file
```

## Dependencies

- **gin-gonic/gin**: HTTP web framework
- **google/uuid**: UUID generation and validation
- **joho/godotenv**: Environment variable loading
- **rs/zerolog**: Structured logging
- **go.mongodb.org/mongo-driver**: MongoDB driver

## Installation & Setup

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose
- MongoDB (if running locally)

### Environment Configuration

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your configuration:
   ```bash
   # Server Configuration
   PORT=8400
   LOGLEVEL=info
   LOGFILE=/root/log/lister.log
   VERSION=v0.1.0

   # MongoDB Configuration
   MONGO_HOST=localhost
   MONGO_PORT=27017
   MONGO_USERNAME=lister_user
   MONGO_PASSWORD=lister_password
   MONGO_DATABASE=poptape_lister
   ```

### Running with Docker Compose

1. Build and start the services:
   ```bash
   docker-compose up --build -d
   ```

2. The API will be available at `http://localhost:1600`
3. MongoDB will be available at `localhost:1601`

### Running Locally

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Make sure MongoDB is running and accessible

3. Build and run the application:
   ```bash
   go build -o lister .
   ./lister
   ```

## Testing

You can test the API using curl or any HTTP client:

```bash
# Check system status
curl http://localhost:1600/list/status

# Get watchlist (requires X-Public-ID header)
curl -H "X-Public-ID: 2a99371f-4188-49b8-a628-85e946540364" \
     http://localhost:1600/list/watchlist

# Add item to watchlist
curl -X POST \
     -H "Content-Type: application/json" \
     -H "X-Public-ID: 2a99371f-4188-49b8-a628-85e946540364" \
     -d '{"uuid":"803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"}' \
     http://localhost:1600/list/watchlist

# Get watching count for an item (no auth required)
curl http://localhost:1600/list/watching/803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12
```

## Data Storage

The application uses MongoDB collections named after the list types:
- `watchlist` - User watchlist items
- `favourites` - Favourite sellers
- `viewed` - Recently viewed items  
- `recentbids` - Recent bid records
- `purchased` - Purchase history

Each document has the structure:
```json
{
    "_id": "user_public_id",
    "list_type": "watchlist", 
    "items": ["uuid1", "uuid2", ...],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
}
```

Lists are limited to 50 items and items are stored in most-recent-first order.

## Notes

- This microservice maintains the latest X number of things for each user
- In production, you should implement proper JWT-based authentication
- Consider implementing proper rate limiting for production use
- The current implementation uses placeholder data for some complex responses (like bid amounts)

## TODO

- Implement proper JWT authentication
- Add comprehensive tests
- Implement pagination for large lists
- Add metrics and monitoring
- Implement proper rate limiting
- Add API documentation (OpenAPI/Swagger)
