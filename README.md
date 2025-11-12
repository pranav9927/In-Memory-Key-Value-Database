
# Redis-like Database Implementation in Go

A high-performance, Redis-compatible database implementation in Go with support for strings, lists, streams, transactions, and blocking operations.

## Features

### Data Types Support

- **Strings**: Basic key-value storage with expiration support (TTL)
- **Lists**: Doubly-ended lists with push/pop operations from both ends
- **Streams**: Append-only log data structure with time-ordered entries and blocking reads


### Core Functionality

- **Transactions**: MULTI/EXEC/DISCARD support for atomic operations
- **Blocking Operations**: XREAD with BLOCK support for real-time stream processing
- **Expiration**: TTL support for keys with automatic cleanup
- **Persistence**: RDB file format support for data durability
- **Type Safety**: Proper type checking with Redis-compatible error messages


### Redis Protocol Compatibility

- RESP (Redis Serialization Protocol) support
- Compatible with standard Redis clients
- Familiar Redis command syntax and behavior




