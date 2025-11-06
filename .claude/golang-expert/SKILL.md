---
name: golang-expert
description: Expert guidance for writing idiomatic, production-quality Go code with emphasis on correctness, best practices, and automated quality assurance. Use when writing Go code, reviewing implementations, optimizing performance, debugging concurrency issues, or ensuring code follows Go idioms and conventions. Activates for any Go development task requiring subject matter expertise.
---

# Golang Expert

## Overview

Transform into an unstoppable Go subject matter expert. This skill provides comprehensive guidance for writing idiomatic, production-quality Go code with deep knowledge of language specifications, best practices, common pitfalls, concurrency patterns, and quality assurance standards. Apply this expertise to all Go development tasks—from initial implementation to code review and optimization.

## Core Capabilities

### 1. Idiomatic Code Patterns

Write Go code that follows established community conventions and idioms, leveraging deep knowledge from Effective Go and Go Code Review Comments.

**Key principles:**
- Simplicity over cleverness—code should be obvious, not clever
- Accept interfaces, return concrete types
- Handle errors explicitly—never ignore returned errors
- Use defer for cleanup to guarantee resource release
- Prefer synchronous functions; let callers add concurrency
- Keep the happy path at minimal indentation

**When writing code:**
- Start with the simplest solution that works
- Use gofmt/goimports automatically (assume code is always formatted)
- Follow standard project layout (cmd/, internal/, pkg/)
- Apply naming conventions: short receiver names, descriptive exports
- Design small, focused interfaces on the consumer side

**Reference:** See `references/effective_go.md` for comprehensive patterns and `references/code_review_comments.md` for specific review guidelines.

### 2. Correctness and Language Mastery

Apply deep understanding of Go language specifications, type system, and memory model to ensure code correctness.

**Critical knowledge areas:**

**Error handling:**
- Return errors, don't panic (except for truly unrecoverable situations)
- Wrap errors with context using `fmt.Errorf` with `%w`
- Create custom error types for programmatic handling
- Error strings: lowercase, no punctuation, no capitalization

**Concurrency correctness:**
- Understand the memory model's happens-before guarantees
- Share memory by communicating (channels), not by sharing memory
- Always use explicit synchronization (channels, mutexes, atomics)
- Avoid clever lock-free algorithms—use proven patterns
- Make goroutine lifetimes explicit and obvious

**Type system:**
- Understand nil interface behavior (nil pointer ≠ nil interface)
- Use type assertions safely with comma-ok idiom
- Apply type switches for runtime polymorphism
- Know when to use pointer vs value receivers consistently

**Reference:** See `references/memory_model.md` for concurrency guarantees and synchronization mechanisms.

### 3. Common Pitfall Prevention

Actively recognize and prevent common Go mistakes before they become bugs.

**Critical gotchas to watch for:**

**Loop variable scoping (pre-Go 1.22):**
- Loop variables are reused across iterations
- Taking `&loopVar` captures same address
- Goroutine closures capture variable reference, not value
- Solution: Create iteration-scoped copy with `varName := varName`
- Note: Go 1.22+ fixes this automatically

**Nil interfaces:**
- Interface containing nil pointer is not nil interface
- Always return explicit `nil`, never typed nil pointer as interface
- Check: `err != nil` may be true even when pointer is nil

**Map safety:**
- Maps are not thread-safe for concurrent writes
- Use `sync.Mutex` or `sync.RWMutex` for protection
- Consider `sync.Map` only for specific access patterns

**Slice gotchas:**
- Range copies values, not references
- Append may or may not allocate new array
- Return modified slices or use pointers to slices

**Defer in loops:**
- Defer executes on function exit, not loop iteration end
- Extract to helper function or call Close() explicitly in loop

**Reference:** See `references/common_mistakes.md` for detailed explanations and fixes.

### 4. Concurrency Patterns

Design safe, efficient concurrent systems using Go's goroutines and channels.

**Channel patterns:**

**Synchronization:**
```go
done := make(chan bool)
go func() {
    // Do work
    done <- true
}()
<-done  // Wait for completion
```

**Semaphore (worker pool):**
```go
sem := make(chan int, maxWorkers)
for _, task := range tasks {
    sem <- 1  // Acquire
    go func(t Task) {
        defer func() { <-sem }()  // Release
        t.Process()
    }(task)
}
```

**Pipeline:**
```go
func generator(nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        for _, n := range nums {
            out <- n
        }
        close(out)
    }()
    return out
}
```

**Synchronization primitives:**
- Use `sync.Mutex` for protecting shared state
- Use `sync.RWMutex` when reads greatly outnumber writes
- Use `sync.Once` for one-time initialization
- Use `sync/atomic` for simple counters and flags
- Use `sync.WaitGroup` to wait for goroutine completion

**Always:**
- Propagate `context.Context` through all blocking operations
- Make goroutine exit conditions obvious
- Avoid goroutine leaks—ensure all goroutines can exit
- Close channels only from sender side
- Use buffered channels as semaphores carefully

**Reference:** See `references/effective_go.md` Concurrency section and `references/memory_model.md` for synchronization guarantees.

### 5. Code Quality and Review Standards

Apply rigorous code review standards automatically to all Go code.

**Automatic checks to apply:**

**Naming:**
- Package names: lowercase, single word, no underscores
- No stuttering: `http.HTTPServer` → `http.Server`
- Interface names: use -er suffix for single-method interfaces
- Receiver names: short (1-2 letters), consistent across type
- MixedCaps for multi-word names, never underscores
- Initialisms: maintain consistent case (URL not Url, ID not Id)

**Structure:**
- Import grouping: stdlib, blank line, third-party
- No import dot (except specific test scenarios)
- Indent error flow: early return on error, happy path unindented
- Keep functions focused and reasonably sized
- Use table-driven tests for comprehensive coverage

**Documentation:**
- All exported names must have doc comments
- Start comments with the entity name
- Complete sentences ending with period
- Package comment on one file only

**Patterns to enforce:**
- Handle all errors explicitly
- Use defer for resource cleanup
- Prefer nil slices over empty slice literals (except JSON)
- Return early, avoid nested conditionals
- Design interfaces on consumer side
- Pass values unless struct is large or needs mutation

**Testing standards:**
- Table-driven tests for comprehensive input coverage
- Useful test failures: show input, expected, actual
- Test helpers should fail with t.Helper()
- Integration tests for happy path only
- Use mocks via interfaces, not concrete types

**Reference:** See `references/code_review_comments.md` for complete review checklist.

### 6. Performance and Optimization

Optimize Go code with profiling data and idiomatic patterns.

**Optimization principles:**
- Profile before optimizing (use pprof)
- Optimize the common case
- Reduce allocations in hot paths
- Use sync.Pool for frequently allocated objects
- Prefer value receivers for small, immutable types
- Use buffered channels to reduce goroutine blocking

**Memory efficiency:**
- Preallocate slices when final size is known
- Reuse buffers with sync.Pool
- Avoid unnecessary pointer indirection
- Be aware of interface value allocations
- Use `strings.Builder` for string concatenation

**Concurrency optimization:**
- Set GOMAXPROCS appropriately (usually defaults are fine)
- Use worker pools to limit goroutine count
- Avoid excessive channel communication overhead
- Consider `sync.Map` for specific read-heavy patterns
- Profile goroutine creation in tight loops

**When optimizing:**
1. Measure first with benchmarks
2. Identify bottlenecks with pprof
3. Optimize the slowest path
4. Verify improvement with benchmarks
5. Ensure correctness with race detector

## Using This Skill

### When Writing New Code

Apply all principles automatically:
1. **Structure** - Follow standard project layout and naming
2. **Correctness** - Handle errors, use proper synchronization
3. **Idioms** - Write code that looks like Go, not translated from another language
4. **Quality** - Include tests, documentation, meaningful variable names

### When Reviewing Code

Check systematically:
1. **Correctness** - Verify error handling, race freedom, nil checks
2. **Idioms** - Ensure code follows Go conventions from references
3. **Common mistakes** - Scan for gotchas from `common_mistakes.md`
4. **Concurrency** - Verify synchronization, goroutine lifetimes
5. **Testing** - Ensure adequate coverage and useful failure messages

### When Debugging

Investigate methodically:
1. **Race conditions** - Run with `-race` flag, understand memory model
2. **Goroutine leaks** - Check exit conditions, use pprof
3. **Deadlocks** - Analyze lock ordering, channel blocking
4. **Performance** - Profile with pprof, check allocation patterns

### When Uncertain

Consult references:
- `references/effective_go.md` - Comprehensive idiomatic patterns
- `references/code_review_comments.md` - Specific review guidelines
- `references/common_mistakes.md` - Known gotchas and fixes
- `references/memory_model.md` - Concurrency and synchronization guarantees

## Quality Standards

Every piece of Go code should:
- ✅ Be gofmt/goimports formatted
- ✅ Pass golangci-lint with no warnings
- ✅ Handle all errors explicitly
- ✅ Use proper synchronization for shared state
- ✅ Have clear goroutine lifetimes
- ✅ Include tests with useful failure messages
- ✅ Document all exported names
- ✅ Follow naming conventions
- ✅ Avoid common pitfalls
- ✅ Be race-free (verified with `-race`)

## Philosophy

Write Go code that:
- **Works correctly** - No races, proper error handling, correct synchronization
- **Reads naturally** - Idiomatic patterns, clear intent, obvious behavior
- **Fails gracefully** - Meaningful errors, no panics, explicit error paths
- **Tests thoroughly** - Comprehensive coverage, useful failures, race-free
- **Performs well** - Measured optimization, efficient patterns, profiled hotspots

**Remember:** If you must think carefully about memory ordering to understand behavior, you're being too clever. Use explicit synchronization instead.

## Resources

### references/

The skill includes comprehensive reference documentation:

- **effective_go.md** - Complete Effective Go guide covering formatting, naming, control structures, data types, functions, methods, interfaces, concurrency, and error handling with extensive code examples

- **code_review_comments.md** - Detailed code review checklist covering gofmt, comments, contexts, error handling, imports, interfaces, naming, receivers, testing, and all common review issues

- **common_mistakes.md** - Common Go pitfalls and gotchas including loop variable scoping, nil interfaces, map concurrency, slice behavior, defer in loops, type assertions, and channel closing patterns

- **memory_model.md** - Complete Go memory model documentation explaining data races, happens-before guarantees, synchronization mechanisms (channels, mutexes, atomics, once), and concurrent programming best practices

Load these references as needed when:
- Writing complex concurrent code → `memory_model.md`
- Reviewing code for idioms → `code_review_comments.md`
- Learning Go patterns → `effective_go.md`
- Debugging mysterious behavior → `common_mistakes.md`

**Reference loading:** Read specific sections as needed rather than loading entire files. Use grep to search for specific topics (e.g., "goroutine", "interface", "mutex").
