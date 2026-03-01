# ADR 0003: Age Library Selection

## Context

We need an encryption library that provides:

- Modern elliptic curve cryptography
- Secure file encryption
- Key generation and management
- Cross-platform compatibility
- Good Go ecosystem support

## Decision

We selected filippo.io/age as our encryption library.

### Why filippo.io/age?

| Aspect | filippo.io/age | golang.org/x/crypto | crypto/twofish |
|--------|----------------|---------------------|----------------|
| API complexity | Low | Medium | High |
| Dependencies | None | None | None |
| Testing | Extensive | Extensive | Minimal |
| Documentation | Good | Good | Fair |
| Maintenance | Active | Stable | Inactive |
| Age format | Native | No | No |
| Go version | Go 1.18+ | Go 1.18+ | Go 1.18+ |

### Comparison with alternatives

#### filippo.io/age vs golang.org/x/crypto/nacl/secretbox

| Factor | age | secretbox |
|--------|-----|-----------|
| Key format | Text-based | Binary |
| Key rotation | Native support | Manual implementation needed |
| File format | Standard (.age) | Custom |
| Error handling | User-friendly | Low-level |
| Deployment | Single dependency | Single dependency |

#### filippo.io/age vs gpg

| Factor | age | gpg |
|--------|-----|-----|
| Dependencies | age binary | Requires gnupg installation |
| Password prompt | No | Yes (by default) |
| Key management | Simple | Complex (public/private keys) |
| Performance | Fast | Slow (for small data) |

## Consequences

### Positive

- Security: Based on X25519 (Curve25519), well-vetted cryptographic primitive
- Usability: Single key file, no password required
- Portability: .age format is cross-platform
- Maintenance: Actively maintained by Filippo Valsorda
- License: BSD-3-Clause (compatible with our MIT license)

### Negative

- Newer library: Less battle-tested than GPG (mitigated by active maintenance)
- Limited format options: Only supports age-specific format

## Implementation Details

### Key Generation

```go
import "filippo.io/age"

identity, err := age.GenerateX25519Identity()
// Returns both private key (identity) and public key (recipient)
```

### Encryption

```go
encryptor, err := age.Encrypt(file, recipient)
io.Copy(encryptor, dataReader)
```

### Decryption

```go
decryptor, err := age.Decrypt(file, identity)
io.Copy(writer, decryptor)
```

## Migration Path

The library provides a migration tool if we ever need to switch:

```bash
# From age to other formats (not needed currently)
age-encrypt --recipient age1... -o output.age input.txt
```

## Status

Accepted - Implemented since v0.1.0

## References

- [filippo.io/age GitHub](https://github.com/FiloSottile/age)
- [go.dev/doc/packages](https://pkg.go.dev/filippo.io/age)
- [Age Format Specification](https://github.com/FiloSottile/age/blob/main/AGE.md)
