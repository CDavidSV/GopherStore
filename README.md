# GopherStore

A lightweight Redis clone written in Go, with support for strings and lists.

## Features

### Data Structures
- **Strings**: Simple key-value pairs with optional expiration
- **Lists**: Ordered collections supporting push/pop operations from both ends

### Key Features
- **RESP Protocol**: Implementation of the Redis Serialization Protocol (RESP)
- **Key Expiration**: TTL support with automatic cleanup of expired keys
- **Concurrent Access**: Thread-safe operations using mutex locks
- **Web Interface**: Web client for testing commands

## Supported Commands

### String Commands

#### SET
Set a key-value pair with optional expiration and conditions.

**Syntax:**
```
SET key value [EX seconds] [PX milliseconds] [NX|XX]
```

**Options:**
- `EX seconds`: Set expiration time in seconds
- `PX milliseconds`: Set expiration time in milliseconds
- `NX`: Only set if key does not exist
- `XX`: Only set if key already exists

**Examples:**
```
SET mykey "Hello"
SET mykey "World" EX 60
SET mykey "Value" NX
SET mykey "Value" PX 5000
```

#### GET
Retrieve the value of a key.

**Syntax:**
```
GET key
```

**Example:**
```
GET mykey
```

**Returns:** The value stored at key, or `nil` if the key does not exist.

### Key Management Commands

#### DEL
Delete one or more keys.

**Syntax:**
```
DEL key [key ...]
```

**Example:**
```
DEL key1 key2 key3
```

**Returns:** Integer representing the number of keys deleted.

#### EXISTS
Check if one or more keys exist.

**Syntax:**
```
EXISTS key [key ...]
```

**Example:**
```
EXISTS key1 key2
```

**Returns:** Integer representing the number of existing keys.

#### EXPIRE
Set a key's time to live in seconds.

**Syntax:**
```
EXPIRE key seconds
```

**Example:**
```
EXPIRE mykey 120
```

**Returns:** `1` if timeout was set, `0` if key does not exist.

#### PEXPIRE
Set a key's time to live in milliseconds.

**Syntax:**
```
PEXPIRE key milliseconds
```

**Example:**
```
PEXPIRE mykey 5000
```

**Returns:** `1` if timeout was set, `0` if key does not exist.

### List Commands

#### LPUSH
Insert values at the head (left) of a list.

**Syntax:**
```
LPUSH key value [value ...]
```

**Example:**
```
LPUSH mylist "world"
LPUSH mylist "hello"
```

**Returns:** Length of the list after the push operation.

#### RPUSH
Insert values at the tail (right) of a list.

**Syntax:**
```
RPUSH key value [value ...]
```

**Example:**
```
RPUSH mylist "hello"
RPUSH mylist "world"
```

**Returns:** Length of the list after the push operation.

#### LPOP
Remove and return the first element from a list.

**Syntax:**
```
LPOP key
```

**Example:**
```
LPOP mylist
```

**Returns:** The value of the first element, or `nil` if the list is empty.

#### RPOP
Remove and return the last element from a list.

**Syntax:**
```
RPOP key
```

**Example:**
```
RPOP mylist
```

**Returns:** The value of the last element, or `nil` if the list is empty.

#### LLEN
Get the length of a list.

**Syntax:**
```
LLEN key
```

**Example:**
```
LLEN mylist
```

**Returns:** Length of the list, or `0` if the key does not exist.

#### LRANGE
Get a range of elements from a list.

**Syntax:**
```
LRANGE key start stop
```

**Example:**
```
LRANGE mylist 0 -1    # Get all elements
LRANGE mylist 0 2     # Get first 3 elements
LRANGE mylist -3 -1   # Get last 3 elements
```

**Returns:** Array of elements in the specified range.

### Connection Commands

#### PING
Test server connection.

**Syntax:**
```
PING [message]
```

**Examples:**
```
PING
PING "Hello World"
```

**Returns:** `PONG` or the provided message.

## Installation & Running

### Prerequisites
- Go 1.21 or higher
- Docker and Docker Compose

### Option 1: Using Docker Compose

1. Clone the repository:

2. Start the services:
```bash
docker-compose up --build
```

This will start two services:
- **GopherStore Server**: Running on port `5001`
- **Web Client**: Accessible at `http://localhost:3000`

### Option 2: Manual Build

1. Clone the repository:
```bash
git clone https://github.com/CDavidSV/GopherStore.git
cd GopherStore
```

2. Start the GopherStore server:
```bash
go run cmd/server/main.go -addr 0.0.0.0:5001
```

3. In a separate terminal, start the web client:
```bash
go run cmd/web/main.go -addr 0.0.0.0:3000
```

The web interface will be available at `http://localhost:3000`

## Configuration

### Server Configuration
The server accepts the following command-line flag:
- `-addr`: Network address to bind to (default: `0.0.0.0:5001`)

### Web Client Configuration
The web client accepts:
- `-addr`: Network address to bind to (default: `0.0.0.0:3000`)

## License

This project is open source and available under the MIT License.
