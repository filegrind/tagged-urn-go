# Tagged URN - Go Implementation

Go implementation of Tagged URN with strict validation, pattern matching, and graded specificity comparison.

## Features

- **Strict Rule Enforcement** - Follows exact same rules as Rust, JavaScript, and Objective-C implementations
- **Case Insensitive** - All input normalized to lowercase (except quoted values)
- **Tag Order Independent** - Canonical alphabetical sorting
- **Special Pattern Values** - `*` (must-have-any), `?` (unspecified), `!` (must-not-have)
- **Value-less Tags** - Tags without values (`tag`) mean must-have-any (`tag=*`)
- **Graded Specificity** - Exact values score higher than wildcards
- **JSON Serialization** - Full JSON marshal/unmarshal support
- **Zero Dependencies** - Only standard library (testify for tests only)

## Installation

```bash
go get github.com/filegrind/tagged-urn-go
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    taggedurn "github.com/filegrind/tagged-urn-go"
)

func main() {
    // Parse a URN
    urn, err := taggedurn.NewTaggedUrnFromString("cap:op=generate;ext=pdf")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Operation:", urn.GetTag("op"))  // "generate"
    fmt.Println("Canonical:", urn.ToString())     // "cap:ext=pdf;op=generate"

    // Build a URN
    built := taggedurn.NewTaggedUrnBuilder("cap").
        Tag("op", "extract").
        Tag("format", "pdf").
        Build()

    // Check matching
    pattern, _ := taggedurn.NewTaggedUrnFromString("cap:op=generate")
    matches, err := urn.Matches(pattern)
    if err != nil {
        log.Fatal(err)
    }
    if matches {
        fmt.Println("URN matches pattern")
    }

    // Get specificity
    fmt.Println("Specificity:", urn.Specificity())
}
```

## API Reference

### TaggedUrn

| Function/Method | Description |
|-----------------|-------------|
| `NewTaggedUrnFromString(s)` | Parse URN from string |
| `NewTaggedUrnFromTags(prefix, tags)` | Create from prefix and tag map |
| `Empty(prefix)` | Create empty URN with prefix |
| `GetTag(key)` | Get value for a tag key |
| `HasTag(key, value)` | Check if tag exists with value |
| `WithTag(key, value)` | Return new URN with tag added/updated |
| `WithoutTag(key)` | Return new URN with tag removed |
| `Matches(pattern)` | Check if URN matches a pattern |
| `CanHandle(request)` | Check if URN can handle a request |
| `Specificity()` | Get graded specificity score |
| `SpecificityTuple()` | Get (exact, mustHaveAny, mustNot) counts |
| `IsMoreSpecificThan(other)` | Compare specificity with another URN |
| `IsCompatibleWith(other)` | Check compatibility with another URN |
| `ToString()` | Get canonical string representation |
| `Hash()` | Get SHA256 hash of canonical form |

### TaggedUrnBuilder

| Method | Description |
|--------|-------------|
| `NewTaggedUrnBuilder(prefix)` | Create builder with prefix |
| `Tag(key, value)` | Add or update a tag (chainable) |
| `Build()` | Build the URN |
| `BuildWithValidation()` | Build with validation (returns error) |

## Matching Semantics

| Pattern | Instance Missing | Instance=v | Instance=x (x≠v) |
|---------|------------------|------------|------------------|
| (missing) or `?` | Match | Match | Match |
| `K=!` | Match | No Match | No Match |
| `K=*` | No Match | Match | Match |
| `K=v` | No Match | Match | No Match |

## Graded Specificity

| Value Type | Score |
|------------|-------|
| Exact value (`K=v`) | 3 |
| Must-have-any (`K=*`) | 2 |
| Must-not-have (`K=!`) | 1 |
| Unspecified (`K=?`) or missing | 0 |

## Error Codes

| Code | Constant | Description |
|------|----------|-------------|
| 1 | `ErrorInvalidFormat` | Empty or malformed URN |
| 2 | `ErrorEmptyTag` | Empty key or value component |
| 3 | `ErrorInvalidCharacter` | Disallowed character in key/value |
| 4 | `ErrorInvalidTagFormat` | Tag not in key=value format |
| 5 | `ErrorMissingPrefix` | URN does not start with prefix |
| 6 | `ErrorDuplicateKey` | Same key appears twice |
| 7 | `ErrorNumericKey` | Key is purely numeric |
| 8 | `ErrorUnterminatedQuote` | Quoted value never closed |
| 9 | `ErrorInvalidEscapeSequence` | Invalid escape in quoted value |
| 10 | `ErrorEmptyPrefix` | Prefix is empty |
| 11 | `ErrorPrefixMismatch` | Prefixes don't match in comparison |

## Testing

```bash
go test -v ./...
```

## Cross-Language Compatibility

This Go implementation produces identical results to:
- [Rust implementation](https://github.com/filegrind/tagged-urn-rs)
- [JavaScript implementation](https://github.com/filegrind/tagged-urn-js)
- [Objective-C implementation](https://github.com/filegrind/tagged-urn-objc)

All implementations pass the same test cases and follow identical rules. See [Tagged URN RULES.md](https://github.com/filegrind/tagged-urn-rs/blob/main/docs/RULES.md) for the complete specification.
