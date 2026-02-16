# REST API Reference

AYB auto-generates REST endpoints for every table in your PostgreSQL database.

## Collections (CRUD)

```
GET    /api/collections/{table}          List records
POST   /api/collections/{table}          Create record
POST   /api/collections/{table}/batch    Batch operations
GET    /api/collections/{table}/{id}     Get record
PATCH  /api/collections/{table}/{id}     Update record (partial)
DELETE /api/collections/{table}/{id}     Delete record
```

### List records

```bash
curl "http://localhost:8090/api/collections/posts?filter=published=true&sort=-created_at&page=1&perPage=20"
```

**Response:**

```json
{
  "items": [
    { "id": 1, "title": "Hello", "published": true, "created_at": "2026-02-07T..." }
  ],
  "page": 1,
  "perPage": 20,
  "totalItems": 42,
  "totalPages": 3
}
```

### Query parameters

| Parameter | Example | Description |
|-----------|---------|-------------|
| `search` | `?search=hello world` | Full-text search across all text columns |
| `filter` | `?filter=status='active' AND age>21` | SQL-safe parameterized filtering |
| `sort` | `?sort=-created_at,+title` | Sort by fields (`-` desc, `+` asc) |
| `page` | `?page=2` | Page number (default: 1) |
| `perPage` | `?perPage=50` | Items per page (default: 20, max: 500) |
| `fields` | `?fields=id,name,email` | Select specific columns |
| `expand` | `?expand=author,category` | Expand foreign key relationships |
| `skipTotal` | `?skipTotal=true` | Skip COUNT query for faster responses |

### Filter syntax

Filters use a SQL-like syntax that is parameterized for safety:

```
# Equality
?filter=status='active'

# Comparison
?filter=age>21
?filter=price<=100

# AND / OR
?filter=status='active' AND category='tech'
?filter=role='admin' OR role='editor'

# NULL checks
?filter=deleted_at IS NULL

# LIKE
?filter=name LIKE '%john%'
```

### Full-text search

Use `?search=` to search across all text columns (`text`, `varchar`, `char`) in a table:

```bash
curl "http://localhost:8090/api/collections/posts?search=postgres database"
```

Search uses PostgreSQL's `websearch_to_tsquery`, so it supports natural search syntax:

```
# Simple search
?search=postgres

# Multi-word (AND by default)
?search=postgres database

# Exact phrase
?search="full text search"

# OR
?search=postgres or mysql

# Exclude terms
?search=postgres -mysql
```

Results are automatically ranked by relevance when no explicit `sort` is provided.

Search can be combined with filters:

```bash
curl "http://localhost:8090/api/collections/posts?search=postgres&filter=published=true&perPage=10"
```

::: tip Performance
For tables with many rows, add a GIN index on text columns for faster search:

```sql
CREATE INDEX posts_fts_idx ON posts USING GIN (
  to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(body, ''))
);
```
:::

### Batch operations

Perform multiple create, update, and delete operations in a single atomic transaction. If any operation fails, all changes are rolled back.

```bash
curl -X POST http://localhost:8090/api/collections/posts/batch \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {"method": "create", "body": {"title": "Post A", "published": true}},
      {"method": "create", "body": {"title": "Post B", "published": false}},
      {"method": "update", "id": "42", "body": {"published": true}},
      {"method": "delete", "id": "99"}
    ]
  }'
```

**Request body:**

| Field | Type | Description |
|-------|------|-------------|
| `operations` | array | Array of operations (max 1000) |
| `operations[].method` | string | `"create"`, `"update"`, or `"delete"` |
| `operations[].id` | string | Record ID (required for update/delete) |
| `operations[].body` | object | Record data (required for create/update) |

**Response** (200 OK):

```json
[
  {"index": 0, "status": 201, "body": {"id": 100, "title": "Post A", "published": true}},
  {"index": 1, "status": 201, "body": {"id": 101, "title": "Post B", "published": false}},
  {"index": 2, "status": 200, "body": {"id": 42, "title": "Existing", "published": true}},
  {"index": 3, "status": 204}
]
```

All operations run in a single database transaction. RLS policies apply. Realtime and webhook events are published after successful commit.

### Create a record

```bash
curl -X POST http://localhost:8090/api/collections/posts \
  -H "Content-Type: application/json" \
  -d '{"title": "New Post", "body": "Content", "published": false}'
```

**Response** (201 Created):

```json
{
  "id": 42,
  "title": "New Post",
  "body": "Content",
  "published": false,
  "created_at": "2026-02-07T22:00:00Z"
}
```

### Get a record

```bash
curl http://localhost:8090/api/collections/posts/42
```

Supports `?fields=` and `?expand=` query parameters.

### Update a record

```bash
curl -X PATCH http://localhost:8090/api/collections/posts/42 \
  -H "Content-Type: application/json" \
  -d '{"published": true}'
```

Only the specified fields are updated (partial update).

### Delete a record

```bash
curl -X DELETE http://localhost:8090/api/collections/posts/42
```

Returns `204 No Content` on success.

### Expand foreign keys

If your `posts` table has an `author_id` column referencing `users(id)`:

```bash
curl "http://localhost:8090/api/collections/posts?expand=author"
```

The response includes the full related record nested under the FK column name.

## Schema

```bash
curl http://localhost:8090/api/schema
```

Returns the full database schema as JSON including tables, columns, types, primary keys, and foreign key relationships.

## Health check

```bash
curl http://localhost:8090/health
```

Returns `200 OK` when the server is running and the database is reachable.

## Error format

All errors return a consistent JSON format:

```json
{
  "message": "collection not found: nonexistent"
}
```

Common HTTP status codes:

| Status | Meaning |
|--------|---------|
| `400` | Invalid request (bad filter syntax, invalid JSON) |
| `401` | Unauthorized (missing or invalid JWT) |
| `404` | Collection or record not found |
| `409` | Conflict (unique constraint violation) |
| `422` | Validation error (NOT NULL violation, check constraint) |
| `500` | Internal server error |
