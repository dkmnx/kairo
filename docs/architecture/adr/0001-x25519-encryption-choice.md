# ADR 0001: X25519 Encryption Choice

## Context

Kairo needs to securely store API keys for multiple providers. The requirements are:

- Encrypt API keys at rest
- Support key rotation
- Cross-platform (Windows, Linux, macOS)
- No external dependencies beyond age
- User-friendly key management

## Decision

We chose **X25519 encryption via the `filippo.io/age` library** for the following reasons:

### Why X25519?

| Factor            | X25519             | RSA                    | NaCl/Box          |
| ----------------- | ------------------ | ---------------------- | ----------------- |
| Key size          | 32 bytes (private) | 2048+ bytes            | 32 bytes          |
| Performance       | Fast               | Slow for large data    | Fast              |
| Simplicity        | Simple API         | Complex key management | Complex API       |
| Library maturity  | Mature (age)       | Legacy                 | Less common in Go |
| User-friendliness | Single key file    | Key pair management    | Binary keys       |

### Why age library?

1. **Simplicity**: Single key file (`age.key`) contains both identity and recipient
2. **No password prompting**: Unlike gpg, age doesn't require interactive password entry
3. **Modern standard**: Based on RFC 7748 X25519
4. **Good Go support**: `filippo.io/age` is actively maintained
5. **Binary format**: `.age` files are well-defined and portable

### Why not alternatives?

- **GPG**: Too complex for CLI tool, requires GPG installation, interactive prompts
- **AES with password**: Password management is error-prone for users
- **NaCl/Box**: Less common in Go ecosystem, more complex key handling

## Consequences

### Positive

- **Security**: X25519 is well-vetted, quantum-resistant (vs RSA)
- **Performance**: Fast encryption/decryption for typical API key sizes
- **User experience**: Single key file, no password required
- **Portability**: `.age` format is standard and well-documented

### Negative

- **Learning curve**: Users unfamiliar with age may need documentation
- **Recovery complexity**: If key is lost, API keys cannot be recovered (user must backup)

## Implementation

```go
// Key generation (internal/crypto/age.go)
key, err := age.GenerateX25519Identity()
```

```go
// Encryption (internal/crypto/age.go)
encryptor, err := age.Encrypt(file, recipient)
```

```go
// Decryption (internal/crypto/age.go)
decryptor, err := age.Decrypt(file, identity)
```

## References

- [age GitHub](https://github.com/FiloSottile/age)
- [filippo.io/age](https://filippo.io/age)
- [RFC 7748 - Elliptic Curves for Security (X25519)](https://datatracker.ietf.org/doc/html/rfc7748)

## Status

**Accepted** - Implemented and in use since v0.1.0
