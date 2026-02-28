<div align="center">
    <img height="200" alt="GoKV" src="https://github.com/user-attachments/assets/8e712e9a-48ad-4ca2-a329-1515cc5b53cb" />
    <p align="center">
        <a href="https://go.dev/"><img alt="Go Version" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white"></a>
        <a href="https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html"><img alt="Architecture" src="https://img.shields.io/badge/Architecture-Clean-green?style=for-the-badge"></a>
        <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/License-MIT-blue?style=for-the-badge"></a>
    </p>

**🔑 Redis-like in-memory key-value store built from scratch in Go**

_TTL support. AOF persistence. Thread-safe. Clean Architecture._

</div>

## ✨ Features

🔑 **11 Commands** - Full implementation of core Redis-like commands  
🔒 **Thread-Safe** - Uses `sync.RWMutex` for safe concurrent access  
⏰ **TTL & Auto-Expiration** - Time-To-Live support with background cleanup goroutine  
💾 **AOF Persistence** - Optional Append-Only File for data durability  
🌐 **TCP Protocol** - Simple text-based protocol compatible with netcat  
🏗️ **Clean Architecture** - Well-organized code with clear separation of concerns  
⚡ **High Performance** - Optimized for high-read scenarios with RWMutex

## 📖 Commands

### Basic

| Command | Syntax | Description |
|---------|--------|-------------|
| `SET` | `SET key value` | Creates or updates a key with the given value |
| `GET` | `GET key` | Retrieves the value of a key |
| `DEL` | `DEL key [key ...]` | Deletes one or more keys |
| `EXPIRE` | `EXPIRE key seconds` | Sets a TTL (Time-To-Live) in seconds |
| `TTL` | `TTL key` | Returns remaining TTL (-1 if no TTL, -2 if not exists) |
| `PERSIST` | `PERSIST key` | Removes TTL, making the key persistent |

### Advanced

| Command | Syntax | Description |
|---------|--------|-------------|
| `KEYS` | `KEYS pattern` | Lists keys matching a pattern (supports `*` and `?` wildcards) |
| `EXISTS` | `EXISTS key [key ...]` | Checks if keys exist, returns count |
| `PING` | `PING [message]` | Returns PONG or echoes the message |
| `INFO` | `INFO` | Returns server statistics |
| `QUIT` | `QUIT` | Closes the connection |

### Categories

• **Write Commands** (persisted to AOF): `SET`, `DEL`, `EXPIRE`, `PERSIST`  
• **Read Commands** (not persisted): `GET`, `TTL`, `KEYS`, `EXISTS`, `PING`, `INFO`

## 🏛️ Architecture

GoKV follows **Clean Architecture** principles, organizing code into four distinct layers with dependencies pointing inward.

```
┌─────────────────────────────────────────┐
│           Adapter Layer                 │
│     TCP Handler + Protocol Parser       │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│          Use Case Layer                 │
│    Command Handler + Stats              │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│           Domain Layer                  │
│   Interfaces + Entities + Commands      │
└─────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────┐
│       Infrastructure Layer              │
│   Store (in-memory) + AOF (persistence) │
└─────────────────────────────────────────┘
```

• **Domain** - Defines contracts (interfaces) and pure entities. No external dependencies  
• **Infrastructure** - Provides concrete implementations for storage and persistence  
• **Adapter** - Handles external I/O, protocol parsing, and response formatting

## 🔐 Thread Safety

GoKV is designed for high concurrency:

• **Read-Write Mutex** - Uses `sync.RWMutex` allowing multiple concurrent readers or exclusive writer access  
• **Background Cleanup** - A dedicated goroutine periodically removes expired keys  
• **Read operations** (`Get`, `Keys`, `Exists`, `TTL`) - Use `RLock()` for concurrent reads  
• **Write operations** (`Set`, `Del`, `Expire`, `Persist`) - Use `Lock()` for exclusive access

## 💾 Persistence (AOF)

Optional Append-Only File for data durability:

• **During Execution** - Write commands are appended to the AOF file  
• **On Startup** - The AOF file is replayed to restore state  
• **Sync After Write** - Each command is synced to disk immediately

**Example AOF file (RESP-like format):**

```
*3
$3
SET
$6
user:1
$4
John
*3
$6
EXPIRE
$6
user:1
$2
60
*2
$3
DEL
$6
user:2
*2
$7
PERSIST
$6
user:1
```

Each entry uses `*N` to indicate the number of arguments, followed by `$len` and the argument value (CRLF-delimited). This format correctly handles values containing spaces.

## 📡 Protocol

Simple text-based protocol over TCP:

**Request:** `COMMAND arg1 arg2 arg3\n`

**Responses:**

| Type | Format | Example |
|------|--------|---------|
| Success | `OK\n` | `OK` |
| Value | `value\n` | `John` |
| Number | `123\n` | `59` |
| Not Found | `nil\n` | `nil` |
| Error | `ERR: message\n` | `ERR: unknown command` |

## 🚀 Getting Started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)

### Build & Run

```bash
git clone https://github.com/rodrigodotdev/gokv.git
cd gokv
go build -o gokv ./cmd/server
./gokv
```

The server listens on port **6379** by default. Connect with any TCP client:

```bash
nc localhost 6379
SET hello world
GET hello
```

### Configuration

GoKV is configured via environment variables or command-line flags. Flags take precedence.

| Setting | Env Var | Flag | Default | Description |
|---------|---------|------|---------|-------------|
| Port | `GOKV_PORT` | `-port` | `6379` | TCP listen port |
| AOF enabled | `GOKV_AOF_ENABLED` | `-aof` | `false` | Enable append-only file persistence |
| AOF file path | `GOKV_AOF_FILE_PATH` | `-aof-file` | `data.aof` | Path to the AOF file |
| Cleanup interval | `GOKV_CLEANUP_INTERVAL_MS` | `-cleanup-interval` | `1000` | Expired-key cleanup sweep interval (ms) |
| Connection timeout | `GOKV_CONN_TIMEOUT_SEC` | `-conn-timeout` | `300` | Idle connection timeout (seconds) |

**Example:**

```bash
GOKV_PORT=6380 GOKV_AOF_ENABLED=true ./gokv
# or
./gokv -port 6380 -aof -conn-timeout 600
```

## 📜 License

This project is licensed under the [MIT License](LICENSE).
