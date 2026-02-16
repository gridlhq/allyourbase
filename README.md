# 👾 Allyourbase

[![CI](https://github.com/gridlhq/allyourbase/actions/workflows/ci.yml/badge.svg)](https://github.com/gridlhq/allyourbase/actions/workflows/ci.yml)
[![Release](https://github.com/gridlhq/allyourbase/actions/workflows/release.yml/badge.svg)](https://github.com/gridlhq/allyourbase/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Status: Beta](https://img.shields.io/badge/Status-Beta-orange)](https://github.com/gridlhq/allyourbase)

Backend-as-a-Service for PostgreSQL. Single binary, zero config, auto-generated REST API + admin dashboard.

```bash
curl -fsSL https://install.allyourbase.io | sh
```

## Quickstart

Start the server (built-in Postgres, nothing else to install):

```bash
ayb start
```

Create a table via SQL:

```bash
ayb sql "CREATE TABLE posts (id serial PRIMARY KEY, title text NOT NULL, body text, created_at timestamptz DEFAULT now())"
```

REST API is auto-generated. Create a record:

```bash
curl -X POST http://localhost:8090/api/collections/posts \
  -H "Content-Type: application/json" \
  -d '{"title": "Hello world", "body": "First post"}'
```

```json
{
  "id": 1,
  "title": "Hello world",
  "body": "First post",
  "created_at": "2025-01-15T10:30:00Z"
}
```

List records:

```bash
curl http://localhost:8090/api/collections/posts
```

```json
{
  "items": [
    {"id": 1, "title": "Hello world", "body": "First post", "created_at": "2025-01-15T10:30:00Z"}
  ],
  "page": 1,
  "perPage": 20,
  "totalItems": 1,
  "totalPages": 1
}
```

Every table gets full CRUD + filtering, sorting, pagination, full-text search.

```bash
# Filter and sort
curl 'http://localhost:8090/api/collections/posts?filter=title="Hello world"&sort=-created_at'

# Admin dashboard
open http://localhost:8090/admin
```

## Point at existing Postgres

```bash
ayb start --database-url postgresql://user:pass@localhost:5432/mydb
```

Schema introspection on startup. Existing tables → instant API endpoints.

## TypeScript SDK

```bash
npm install @allyourbase/js
```

```typescript
import { AYBClient } from "@allyourbase/js";

const ayb = new AYBClient("http://localhost:8090");

// CRUD with filters, sort, pagination
const { items } = await ayb.records.list("posts", {
  filter: "published=true",
  sort: "-created_at",
  expand: "author",
  perPage: 50,
});
const post = await ayb.records.create("posts", { title: "Hello" });

// Auth
await ayb.auth.login("user@example.com", "password");
const { user } = await ayb.auth.signInWithOAuth("google");

// Realtime
ayb.realtime.subscribe(["posts"], (e) => console.log(e.action, e.record));
```

## Features

- **REST API** — auto-generated CRUD for every table (filter, sort, paginate, full-text search, FK expand)
- **Admin dashboard** — SQL editor, API explorer, schema browser, RLS manager, storage browser, user management
- **Auth** — email/password, JWT, OAuth (Google/GitHub), email verify, password reset
- **Storage** — local disk or S3-compatible (R2, MinIO, etc.), signed URLs
- **Realtime** — Server-Sent Events, RLS-filtered table subscriptions
- **RLS** — JWT claims → Postgres session vars for row-level security policies
- **RPC** — call Postgres functions via `POST /api/rpc/{function}`
- **Type generation** — `ayb types typescript` generates types from schema
- **Embedded Postgres** — built-in, auto-downloaded, zero external dependencies
- **MCP server** — `ayb mcp` for AI tool integration (Claude Code, Cursor, Windsurf)

## Config (optional)

Zero config by default. Customize via `ayb.toml`:

```toml
[server]
port = 8090

[database]
url = "postgresql://user:pass@localhost:5432/mydb"

[auth]
enabled = true

[storage]
enabled = true
backend = "s3"
```

Precedence: **defaults → ayb.toml → env vars (`AYB_` prefix) → CLI flags**. Check resolved config: `ayb config`.

## Migrating from other platforms

```bash
# From PocketBase
ayb start --from ./pb_data

# From Supabase
ayb migrate supabase --source-url postgres://...supabase... --database-url postgres://localhost/mydb

# From Firebase
ayb migrate firebase --auth-export users.json --firestore-export firestore.json --database-url postgres://localhost/mydb
```

Migrations import users (with password hashes), data, OAuth providers, Firestore documents, and RLS policies.

## CLI

```
ayb start        Start the server (embedded or external Postgres)
ayb stop         Stop the server
ayb status       Show server status
ayb init <name>  Scaffold a new project
ayb config       Print resolved config
ayb migrate      Run migrations or import from another platform
ayb sql          Execute SQL queries
ayb schema       Inspect database schema
ayb types        Generate TypeScript types
ayb logs         View server logs
ayb uninstall    Remove AYB from your system
```

## Comparison

| | PocketBase | Supabase (self-hosted) | AllYourBase |
|---|---|---|---|
| Database | SQLite | PostgreSQL | PostgreSQL |
| Deployment | Single binary | 10+ Docker containers | Single binary |
| Configuration | One file | Dozens of env vars | One file (or none) |
| Row-level security | No | Yes | Yes |
| Docker required | No | Yes | No |

## Deployment

See [deployment guide](docs-site/guide/deployment.md) for systemd, Docker, and other deployment options.

## Install options

```bash
# Specific version
curl -fsSL https://install.allyourbase.io | sh -s -- v0.1.0

# Custom directory
AYB_INSTALL=/opt/ayb curl -fsSL https://install.allyourbase.io | sh

# From source
git clone https://github.com/gridlhq/allyourbase.git
cd allyourbase && make build

# Uninstall
ayb uninstall
```


## License

[MIT](LICENSE)
