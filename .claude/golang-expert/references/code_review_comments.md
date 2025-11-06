# Go Code Review Comments - Complete Reference

This page collects common code review comments so that detailed explanations can be referred to by a shorthand note. It supplements Effective Go and is organized alphabetically by topic.

## Gofmt

Run `gofmt` on all code to fix mechanical style issues automatically. Most Go codebases use this tool. For enhanced functionality, `goimports` additionally manages import statements.

## Comment Sentences

All comments documenting declarations should be complete sentences. This ensures proper formatting when extracted into godoc.

**Guidelines:**
- Begin with the entity name
- End with a period
- Example: `// Request represents a request to run a command.`
- Example: `// Encode writes the JSON encoding of req to w.`

## Contexts

Values of `context.Context` carry security credentials, tracing information, deadlines, and cancellation signals across API boundaries.

**Best practices:**
- Accept Context as the first function parameter: `func F(ctx context.Context, /* others */) {}`
- Pass Context explicitly through call chains, even when uncertain
- Never add Context as a struct field; pass as a method parameter instead
- Exception: methods matching standard library or third-party interfaces
- Avoid custom Context types or non-Context interfaces
- Pass application data via parameters, receivers, globals, or Context values
- Contexts are immutable—safe to share across calls with identical deadlines

## Copying

Copying structs from other packages can create unexpected aliasing issues.

**Rule:** Do not copy type `T` if its methods are associated with pointer type `*T`. Example: copying `bytes.Buffer` causes the slice in the copy to alias the original's array, producing surprising behavior.

## Crypto Rand

Never use `math/rand` or `math/rand/v2` for cryptographic keys, even temporary ones.

**Why:** Seeded with `Time.Nanoseconds()`, entropy is limited.

**Solution:** Use `crypto/rand.Reader` for bytes. For text, use `crypto/rand.Text()`, or encode random bytes with `encoding/hex` or `encoding/base64`.

```go
import (
    "crypto/rand"
    "fmt"
)

func Key() string {
  return rand.Text()
}
```

## Declaring Empty Slices

Prefer nil slice declaration over empty literal initialization.

**Preferred:**
```go
var t []string
```

**Avoid:**
```go
t := []string{}
```

Both have zero length and capacity, but nil slices are idiomatic. The nil slice is the standard Go style.

**Exception:** Use non-nil, zero-length slices when encoding JSON (nil encodes to `null`; `[]string{}` encodes to JSON array `[]`).

## Doc Comments

**Requirements:**
- All top-level, exported names must have doc comments
- Include non-trivial unexported type or function declarations
- Follow standard Go commentary conventions

See Effective Go for detailed guidance.

## Don't Panic

Do not use `panic` for normal error handling. Instead, return `error` as an additional return value.

## Error Strings

Error strings follow specific formatting rules:

**Guidelines:**
- Do not capitalize (unless proper nouns or acronyms begin the message)
- Do not end with punctuation
- Rationale: errors typically print following context

**Correct:**
```go
fmt.Errorf("something bad")
```

**Avoid:**
```go
fmt.Errorf("Something bad")
```

This prevents spurious capitals mid-message when logged: `log.Printf("Reading %s: %v", filename, err)`.

Note: Logging, which is line-oriented, is exempt from this rule.

## Examples

New packages should include usage examples: a runnable `Example()` function or simple test demonstrating complete call sequences.

See the Go blog on testable Example() functions.

## Goroutine Lifetimes

Make goroutine exit conditions explicit.

**Problems with unclear lifetimes:**
- Goroutines leak when blocking on unreachable channels (garbage collector won't terminate them)
- Sends on closed channels cause panics
- Modifying in-flight inputs creates data races
- Indefinite goroutines produce unpredictable memory usage

**Solution:** Keep concurrent code simple enough that lifetimes are obvious. Document exit conditions when complexity is unavoidable.

## Handle Errors

Do not discard errors using `_` variables.

**Obligations:**
- Check all returned errors
- Verify function success
- Handle the error, return it, or panic (only in truly exceptional situations)

## Imports

Organize imports in groups separated by blank lines.

**Structure:**
1. Standard library packages (always first)
2. Blank line
3. Third-party packages

**Example:**
```go
package main

import (
    "fmt"
    "hash/adler32"
    "os"

    "github.com/foo/bar"
    "rsc.io/goversion/version"
)
```

**Naming:** Avoid renaming imports except to prevent collisions. Good package names should not require renaming. When renaming necessary, rename the most local or project-specific import.

Use `goimports` to automate this organization.

## Import Blank

Packages imported only for side effects use syntax: `import _ "pkg"`.

**Rule:** Only use in the main package or tests requiring them.

## Import Dot

The `import .` form has one acceptable use case:

Tests with circular dependencies that cannot be part of the tested package:

```go
package foo_test

import (
    "bar/testutil" // also imports "foo"
    . "foo"
)
```

The test file cannot be in package `foo` (uses `bar/testutil`, which imports `foo`), so `import .` allows the test to operate as if part of `foo`.

**General rule:** Avoid `import .` in programs—it obscures whether identifiers are top-level or imported, harming readability.

## In-Band Errors

Avoid functions signaling errors via specific return values (like -1 or null).

**Problem:**
```go
// Lookup returns "" if no mapping exists
func Lookup(key string) string

Parse(Lookup(key))  // Unclear if "" is valid or an error
```

**Solution:** Use multiple return values with error or boolean status:

```go
// Lookup returns the value and ok=false if missing
func Lookup(key string) (value string, ok bool)

Parse(Lookup(key))  // Compile-time error—forces correct handling
```

**Correct usage:**
```go
value, ok := Lookup(key)
if !ok {
    return fmt.Errorf("no value for %q", key)
}
return Parse(value)
```

**Exception:** Return values like `nil`, `""`, `0`, `-1` are acceptable when they're legitimate results the caller doesn't handle differently.

## Indent Error Flow

Keep normal code path at minimal indentation; indent error handling first.

**Avoid:**
```go
if err != nil {
    // error handling
} else {
    // normal code
}
```

**Preferred:**
```go
if err != nil {
    // error handling
    return
}
// normal code
```

**With initialization:**
```go
// Wrong nesting:
if x, err := f(); err != nil {
    return
} else {
    // use x
}

// Better—separate the declaration:
x, err := f()
if err != nil {
    return
}
// use x
```

## Initialisms

Words that are initialisms or acronyms maintain consistent case throughout.

**Rules:**
- "URL" appears as "URL" or "url", never "Url"
- "ID" (identifier) appears as "ID" or "id", never "Id"
- Multiple initialized words: `xmlHTTPRequest` or `XMLHTTPRequest`

**Examples:**
- `ServeHTTP` (not `ServeHttp`)
- `appID` (not `appId`)

**Exemption:** Code generated by protocol buffer compilers is exempt.

## Interfaces

Design interfaces on the consuming side, not the implementing side.

**Principles:**
- Interfaces belong in the package using those interface types
- Implementing packages return concrete types (usually pointers or structs)
- This allows adding methods without extensive refactoring

**Antipattern—defining for mocking:**
```go
// DO NOT DO THIS:
package producer

type Thinger interface { Thing() bool }

type defaultThinger struct{ … }
func (t defaultThinger) Thing() bool { … }
func NewThinger() Thinger { return defaultThinger{ … } }
```

**Correct approach:**
```go
// In producer:
type Thinger struct{ … }
func (t Thinger) Thing() bool { … }
func NewThinger() Thinger { return Thinger{ … } }

// In consumer tests:
type fakeThinger struct{ … }
func (t fakeThinger) Thing() bool { … }
```

**Rule:** Do not define interfaces before realistic usage examples. Without knowing how an interface is used, determining necessity and required methods is premature.

## Line Length

No rigid line length limit exists, but avoid uncomfortable lengths.

**Guidelines:**
- Do not break lines artificially to meet length targets if more readable long
- Break lines for semantic reasons, not purely for length
- If lines become too long, reconsider parameter count, variable names, or semantics
- This principle mirrors function length guidance: break at boundaries for clarity, not arbitrary line counts

## Mixed Caps

Names use mixed capitals—this applies uniformly across Go, even when conflicting with other language conventions.

**Rules:**
- Exported names: `MixedCaps`
- Unexported names: `mixedCaps` (not `mixed_caps` or `MIXED_CAPS`)
- Constants follow the same pattern: unexported constant is `maxLength`, not `MaxLength` or `MAX_LENGTH`

See also: Initialisms section.

## Named Result Parameters

Named result parameters appear repetitive in godoc. Consider whether naming adds clarity.

**Avoid unnecessary naming:**
```go
// Repetitive in documentation:
func (n *Node) Parent1() (node *Node) {}
func (n *Node) Parent2() (node *Node, err error) {}

// Better:
func (n *Node) Parent1() *Node {}
func (n *Node) Parent2() (*Node, error) {}
```

**Use naming when helpful:**
```go
// Clearer with names:
func (f *Foo) Location() (lat, long float64, err error)
```

Than without:
```go
func (f *Foo) Location() (float64, float64, error)
```

**Guidance:** Name parameters to enhance godoc clarity, not to enable naked returns. Clarity always outweighs saving one or two lines.

**Exception:** Name parameters when needed to modify them in deferred closures—always acceptable.

## Naked Returns

A `return` statement without arguments returns named return values.

```go
func split(sum int) (x, y int) {
    x = sum * 4 / 9
    y = sum - x
    return  // returns x, y
}
```

See Named Result Parameters for guidance on when to use.

## Package Comments

Package comments must appear adjacent to the package clause with no blank line between them.

**Format:**
```go
// Package math provides basic constants and mathematical functions.
package math
```

```go
/*
Package template implements data-driven templates for generating textual
output such as HTML.
....
*/
package template
```

**For `package main`:**
After the binary name, capitalization is flexible. Acceptable patterns:

- `// Binary seedgen ...`
- `// Command seedgen ...`
- `// Program seedgen ...`
- `// The seedgen command ...`
- `// The seedgen program ...`
- `// Seedgen ...`

When the binary name is first, capitalize it even if unconventional for the command invocation—these are public documentation requiring proper English.

**Avoid:** Starting with lowercase words in package comments.

## Package Names

Package names appear as a prefix to all references. Omit the package name from identifiers.

**Example:**
In package `chubby`:
- **Avoid:** `type ChubbyFile` (redundant—clients write `chubby.ChubbyFile`)
- **Prefer:** `type File` (clients write `chubby.File`)

**Avoid meaningless names:** `util`, `common`, `misc`, `api`, `types`, `interfaces`.

See Effective Go and the Go blog on package names.

## Pass Values

Do not pass pointers as function arguments solely to save bytes.

**Rule:** If a function refers to argument `x` only as `*x` throughout, the argument shouldn't be a pointer.

**Common cases:**
- Pointers to strings (`*string`)
- Pointers to interface values (`*io.Reader`)

Both have fixed size and pass directly without pointers.

**Exception:** This advice does not apply to large structs or small structs that might grow.

## Receiver Names

Method receiver names should reflect identity—typically a one or two-letter abbreviation.

**Examples:**
- `c` for "Client"
- `cl` for "Client" (if single letter causes collision)

**Avoid:** Generic names like "me", "this", "self" (typical in OOP languages but inappropriate in Go).

**Principle:** Receivers are parameters like any other; name accordingly.

**Guidelines:**
- Names need not be fully descriptive—role is obvious
- Can be very short; appears frequently
- Be consistent—use the same abbreviation across all methods of a type

## Receiver Type

Choosing value versus pointer receivers is challenging. Detailed guidelines:

**Use pointer receivers when:**
- Method must mutate the receiver
- Receiver contains `sync.Mutex` or similar synchronization fields
- Receiver is a large struct or array (assume it's too large if passing all elements as arguments would be excessive)
- Changes must be visible in the original receiver
- Receiver is a struct, array, or slice with pointer elements that might be mutating

**Use value receivers when:**
- Receiver is a map, func, or chan (don't use pointers to these)
- Receiver is a slice and the method doesn't reslice or reallocate
- Receiver is small, naturally a value type (like `time.Time`)
- No mutable fields and no pointers present
- Receiver is a basic type like `int` or `string`

**Additional consideration:** Value receivers can reduce garbage by using on-stack copies instead of heap allocation (though compilers try to optimize this).

**Consistency rule:** Do not mix receiver types. Choose pointer or value receivers for all methods on a type.

**Default:** When uncertain, use a pointer receiver.

## Synchronous Functions

Prefer synchronous functions (returning results directly or finishing operations before returning) over asynchronous ones.

**Benefits:**
- Goroutines stay localized within calls
- Lifetimes are easier to reason about
- Leak and data-race avoidance
- Simpler testing—pass input, check output; no polling needed

**Concurrency addition:** Callers needing concurrency easily invoke synchronous functions from separate goroutines.

**Challenge:** Removing unnecessary asynchronous concurrency from the caller side is difficult or impossible.

## Useful Test Failures

Tests should fail with clear, helpful messages.

**Requirements:**
- State what was wrong
- Include inputs
- Show actual output
- Show expected output
- Assume the debugger is not the test author

**Pattern:**
```go
if got != tt.want {
    t.Errorf("Foo(%q) = %d; want %d", tt.in, got, tt.want)
}
```

Note the order: `actual != expected`, with the error message mirroring this sequence.

**Techniques:**
- Write descriptive helper output
- Consider table-driven tests
- Wrap test helpers with distinct TestFoo functions for clarity on failure:

```go
func TestSingleValue(t *testing.T) { testHelper(t, []int{80}) }
func TestNoValues(t *testing.T)    { testHelper(t, []int{}) }
```

**Responsibility:** Provide helpful messages for whoever debugs your code later.

## Variable Names

Variable names should be short, especially for local variables with limited scope.

**Preference scale:**
- Prefer `c` to `lineCount`
- Prefer `i` to `sliceIndex`

**Core principle:** The further from declaration a name is used, the more descriptive it must be.

**Guidance by context:**
- Method receiver: one or two letters suffice
- Common loop variables and readers: single letter (`i`, `r`)
- Unusual or global variables: more descriptive names

See the Google Go Style Guide for extended discussion.
