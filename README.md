# Caddy GoatCounter Module

[![CI](https://github.com/jarv/caddy-goatcounter/actions/workflows/ci.yml/badge.svg)](https://github.com/jarv/caddy-goatcounter/actions/workflows/ci.yml)

A Caddy v2 module that provides server-side tracking for GoatCounter analytics. Perfect for tracking RSS feeds, API endpoints, and other server-side requests that don't execute JavaScript.

## Features

- **Server-side tracking** - Works for RSS readers and other clients that don't execute JavaScript
- **Asynchronous** - Doesn't block request processing
- **Configurable paths** - Track specific endpoints or all requests
- **Preserves visitor info** - Forwards User-Agent and IP address data
- **Lightweight** - Minimal overhead with configurable timeouts

## Installation

### Method 1: Using xcaddy (Recommended)

Build Caddy with the GoatCounter module:

```bash
xcaddy build --with github.com/jarv/caddy-goatcounter
```

### Method 2: Build from source

1. Clone this repository
2. Build Caddy with the module:

```bash
xcaddy build --with github.com/jarv/caddy-goatcounter=./
```

## Configuration

### Caddyfile

Add the `goatcounter` directive to your Caddyfile:

```caddyfile
example.com {
    goatcounter {
        api_host localhost:5000
        site mysite.goatcounter.com
        paths /rss /api/feeds
        user_agent "MyApp/1.0"
    }

    file_server
}
```

### JSON Configuration

```json
{
    "handler": "goatcounter",
    "api_host": "localhost:5000",
    "site": "mysite.goatcounter.com",
    "paths": ["/rss", "/api/feeds"],
    "user_agent": "MyApp/1.0"
}
```

## Directive Options

- `api_host` (required): Your GoatCounter API host (e.g., "localhost:5000" or "api.goatcounter.com")
- `site` (required): Your GoatCounter site domain (e.g., "mysite.goatcounter.com")
- `token` (optional): Bearer token for authentication (if required by your GoatCounter instance)
- `paths` (optional): List of path prefixes to track. If omitted, all requests are tracked
- `user_agent` (optional): Custom User-Agent for tracking requests. Defaults to "caddy-goatcounter/1.0"

## Examples

### Track All Requests

```caddyfile
example.com {
    goatcounter {
        api_host localhost:5000
        site mysite.goatcounter.com
    }

    file_server
}
```

### Track Specific Paths Only

```caddyfile
api.example.com {
    goatcounter {
        api_host api.goatcounter.com
        site myapi.goatcounter.com
        paths /rss /feeds /api/v1
        token "your-api-token"
    }

    reverse_proxy backend:8080
}
```

### Custom Configuration

```caddyfile
blog.example.com {
    goatcounter {
        api_host localhost:5000
        site myblog.goatcounter.com
        paths /rss.xml /atom.xml
        user_agent "BlogTracker/2.0"
    }

    file_server
}
```

## How It Works

1. The module intercepts HTTP requests matching configured paths
2. Asynchronously sends tracking data to your GoatCounter API using POST `/api/v0/count`
3. Sends comprehensive visitor information in JSON format:
   - Request path and query parameters
   - User-Agent header
   - Referer header
   - Client IP address (via X-Forwarded-For or X-Real-IP)
   - Timestamp in ISO 8601 format
4. Continues processing the original request without delay
5. Uses proper GoatCounter API v0 schema for maximum compatibility

## Testing with Docker Compose

Use the included Docker Compose setup to test the module:

```bash
# Build and start the test environment
docker-compose up --build

# Test tracking
curl -H "User-Agent: TestBot/1.0" http://localhost:8080/rss
curl -H "User-Agent: FeedReader/2.0" http://localhost:8080/api/feeds
```

## GoatCounter Setup

1. Create a GoatCounter account at https://goatcounter.com
2. Note your site domain (e.g., `mysite.goatcounter.com`)
3. For self-hosted GoatCounter instances, use your API host (e.g., `localhost:5000`)
4. If using authentication, generate an API token from your GoatCounter settings
5. Configure the module with your `api_host`, `site`, and optionally `token`

## Troubleshooting

### Enable Debug Logging

Add logging configuration to see tracking requests:

```caddyfile
{
    log {
        level DEBUG
    }
}

example.com {
    goatcounter {
        api_host localhost:5000
        site mysite.goatcounter.com
        paths /rss
    }

    file_server
}
```

### Common Issues

1. **Tracking not working**: Verify your `api_host` and `site` configuration are correct
2. **High latency**: The module uses a 3-second timeout for tracking requests
3. **Path not tracked**: Ensure the request path matches your configured paths
4. **Authentication errors**: Add a valid `token` if your GoatCounter instance requires authentication

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test with the Docker Compose setup
5. Submit a pull request

## Support

- Report issues on GitHub
- Check GoatCounter documentation for analytics setup
- Ensure your GoatCounter instance allows server-side tracking