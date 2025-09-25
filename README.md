![All tests pass](https://github.com/cliveyg/poptape-lister-redux/actions/workflows/testsuite.yml/badge.svg) ![Tests passed](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cliveyg/d3496d4edcc5b63763adc9f85d5c63e2/raw/2cc2682b1674439bf153f66431428e05cb9514fe/poptape-lister-redux-go-tests.json&label=Tests) ![Test coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/cliveyg/d3496d4edcc5b63763adc9f85d5c63e2/raw/2cc2682b1674439bf153f66431428e05cb9514fe/poptape-lister-redux-go-coverage.json&label=Test%20Coverage) ![Release](https://img.shields.io/github/v/release/cliveyg/poptape-lister-redux)

# poptape-lister-redux

Golang microservice for Poptape Auctions system.  List management: recently viewed, watchlists, watchers of items, etc.

This is a Go implementation of the [original Python poptape-lister](https://github.com/cliveyg/poptape-lister) microservice.

See [this gist](https://gist.github.com/cliveyg/cf77c295e18156ba74cda46949231d69) for an overview of how this microservice fits into the auction system.

## Features

- **MongoDB Integration**: Persistent data storage
- **Gin-Gonic Router**: Fast HTTP web framework
- **UUID Support**: Strong validation with `google/uuid`
- **Dockerized**: Docker setup with MongoDB
- **Structured Logging**: Uses zerolog for logs
- **Middleware Support**: Authentication, CORS, rate limiting, JSON validation

## API Routes

### Authenticated Routes

All authenticated routes require an `x-access-token` header containing a JWT token. The token is passed to an external API for authorization.

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
Returns:
```json
{
  "message": "Created"
}
```

```
DELETE /list/watchlist/:itemId
```
Removes an item from the user's watchlist.

Returns:
```json
{}
```

```
DELETE /list/watchlist
```
Removes all items from the user's watchlist.

Returns:
```json
{}
```

#### Recently Viewed Items

```
GET /list/viewed
```
Returns a list of recently viewed item UUIDs.

Example response:
```json
{
  "viewed": [
    "2a99371f-4188-49b8-a628-85e946540364",
    "803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
  ]
}
```

```
POST /list/viewed
```
Adds an item to the recently viewed list.

#### Favourite Sellers

```
GET /list/favourites
```
Returns a list of favourite seller UUIDs.

Example response:
```json
{
  "favourites": [
    "2a99371f-4188-49b8-a628-85e946540364",
    "803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"
  ]
}
```

```
POST /list/favourites
DELETE /list/favourites/:itemId
DELETE /list/favourites
```
Add/remove favourite sellers (by their UUID).

#### Recent Bids

```
GET /list/recentbids
```
Returns a list of recent bid UUIDs.

Example response:
```json
{
  "recentbids": [
    "a47cdbb5-2e45-4aef-af71-82736351f049",
    "2a99371f-4188-49b8-a628-85e946540364"
  ]
}
```

#### Purchase History

```
GET /list/purchased
```
Returns a list of purchased item UUIDs.

Example response:
```json
{
  "purchased": [
    "a933d845-bf82-421c-bf5c-57f81c182912",
    "a47cdbb5-2e45-4aef-af71-82736351f049"
  ]
}
```

### Public Routes

#### Watching Count

```
GET /list/watching/:item_id
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

## Testing

You can test the API using curl or any HTTP client:

```bash
# Check system status
curl http://localhost:1600/list/status

# Get watchlist (requires x-access-token header with JWT)
curl -H "x-access-token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:1600/list/watchlist

# Add item to watchlist
curl -X POST \
     -H "Content-Type: application/json" \
     -H "x-access-token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     -d '{"uuid":"803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12"}' \
     http://localhost:1600/list/watchlist

# Remove item from watchlist
curl -X DELETE \
     -H "x-access-token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:1600/list/watchlist/803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12

# Remove all items from watchlist
curl -X DELETE \
     -H "x-access-token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     http://localhost:1600/list/watchlist

# Get watching count for an item (no auth required)
curl http://localhost:1600/list/watching/803be8ad-fe4b-4fb2-b8d8-fe9fcedfbb12
```

## Data Storage

MongoDB collections are named after the list types:
- `watchlist` - User watchlist items (UUID string array)
- `favourites` - Favourite sellers (UUID string array)
- `viewed` - Recently viewed items (UUID string array)
- `recentbids` - Recent bid records (UUID string array)
- `purchased` - Purchase history (UUID string array)

Each document structure:
```json
{
  "_id": "public_id",
  "item_ids": ["uuid1", "uuid2", ...],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```
Lists are limited to 50 items, stored in most-recent-first order.

## Notes

- This microservice maintains the latest X number of things for each user
- In production, use a JWT for `x-access-token` and external authorization
- Consider implementing proper rate limiting for production use

## TODO

- ~~Implement JWT authentication~~
- ~~Add comprehensive tests~~
- ~~Implement pagination for large lists~~
- Add metrics and monitoring
- Implement proper rate limiting
- Add API documentation (OpenAPI/Swagger)

## License

This project is licensed under the terms of the GNU General Public License v3.0 (GPL-3.0).  
See the [LICENSE](./LICENSE) file for details.