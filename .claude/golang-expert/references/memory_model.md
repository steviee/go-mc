# The Go Memory Model

## Overview

The Go Memory Model specifies the conditions under which reads of a variable in one goroutine can be guaranteed to observe values produced by writes to the same variable in a different goroutine.

**Core principle:** Programs that modify data being simultaneously accessed by multiple goroutines must serialize such access.

To serialize access, protect the data with channel operations or other synchronization primitives such as those in the `sync` and `sync/atomic` packages.

## Advice

**If you must read the rest of this document to understand the behavior of your program, you are being too clever. Don't be clever.**

Use synchronization primitives (channels, mutexes, atomics) rather than relying on subtle memory ordering guarantees.

## Data Races

A **data race** is defined as:
- A write to a memory location happening concurrently with another read or write to that same memory location
- Unless all the accesses are atomic operations provided by the `sync/atomic` package

**Data races are undefined behavior.** Implementations may report some, but not finding any races does not guarantee race-freedom.

**Race-free programs:**
Programs with no data races are **sequentially consistent**—they behave as if all goroutines were multiplexed onto a single processor.

## Formal Memory Model

The memory model defines requirements for Go implementations through three concepts:

### 1. Happens Before

Within a single goroutine, reads and writes must behave as if they executed in the order specified by the program. However, the compiler and processor may reorder operations as long as the reordering doesn't change behavior within that goroutine.

Because of reordering, execution order observed by one goroutine may differ from the order perceived by another. For example:

```go
var a, b int

// Goroutine 1
a = 1
b = 2

// Goroutine 2
print(b)  // Might print: 2
print(a)  // Then print: 0 (if b=2 is observed before a=1)
```

The **happens before** relation specifies when one memory operation is guaranteed to happen before another.

**Rule:** If event `e1` happens before event `e2`, then `e2` happens after `e1`. If `e1` neither happens before nor after `e2`, they happen concurrently.

### 2. Synchronization

**Read observation rules:**

A read `r` of a variable `v` is allowed to observe a write `w` to `v` if:
1. `r` does not happen before `w`
2. There is no other write `w'` to `v` that happens after `w` but before `r`

**Guaranteed observation:**

To guarantee that a read `r` observes a specific write `w` to `v`:
1. `w` must happen before `r`
2. Any other write to `v` must happen before `w` or after `r`

This second condition is stronger—it requires no concurrent writes.

**Single goroutine:**
Within a single goroutine, there is no concurrency, so these two conditions are equivalent. The read `r` observes the value written by the most recent write `w` to `v`.

**Multiple goroutines:**
When multiple goroutines access a shared variable `v`, they must use synchronization events to establish happens-before conditions ensuring reads observe intended writes.

### 3. Sequential Consistency

**Zero value initialization:**
The initialization of variable `v` with the zero value for `v`'s type happens before the first use of `v`.

**Reads of values larger than single machine word:**
These behave as multiple machine-word-sized operations in unspecified order.

## Synchronization Mechanisms

### Program Initialization

**Package initialization:**
- Program initialization runs in a single goroutine
- `init` functions run sequentially
- Package `p` importing package `q`: `q`'s `init` functions happen before `p`'s

**Main function:**
The start of function `main` happens after all `init` functions finish.

### Goroutine Creation

The `go` statement starting a new goroutine happens before the goroutine's execution begins.

**Example:**
```go
var a string

func f() {
    print(a)
}

func hello() {
    a = "hello, world"
    go f()  // Will observe "hello, world" (or any subsequent value of a)
}
```

The assignment to `a` happens before `go f()`, which happens before `f()` executes.

### Goroutine Destruction

The exit of a goroutine is **not guaranteed** to happen before any event in the program.

**Problematic example:**
```go
var a string

func hello() {
    go func() { a = "hello" }()
    print(a)  // Not guaranteed to observe "hello"
}
```

The assignment to `a` is not followed by any synchronization event. There's no guarantee the assignment will be observed by any other goroutine. A compiler might optimize away the entire `go` statement.

**Fix: Use explicit synchronization:**
```go
var a string
var done = make(chan bool)

func hello() {
    go func() {
        a = "hello"
        done <- true
    }()
    <-done
    print(a)  // Guaranteed to observe "hello"
}
```

### Channel Communication

Channel communication is the main method of synchronization between goroutines.

**Sends and receives:**
- A send on a channel happens before the corresponding receive from that channel completes

**Unbuffered channels:**
- A receive from an unbuffered channel happens before the send on that channel completes

**Buffered channels:**
- The `k`th receive on a channel with capacity `C` happens before the `k+C`th send completes

**Example (unbuffered):**
```go
var c = make(chan int)
var a string

func f() {
    a = "hello, world"
    c <- 0  // Send
}

func main() {
    go f()
    <-c     // Receive
    print(a)  // Guaranteed to print "hello, world"
}
```

The write to `a` happens before the send on `c`, which happens before the receive completes, which happens before the `print`.

**Example (buffered):**
```go
var c = make(chan int, 1)
var a string

func f() {
    a = "hello, world"
    <-c  // Receive (k-th)
}

func main() {
    c <- 0   // Send ((k+C)-th where C=1)
    go f()
    print(a) // NOT guaranteed to observe "hello, world"
}
```

This is not guaranteed because the send completes before the receive.

**Example (semaphore pattern):**
```go
var limit = make(chan int, 3)

func main() {
    for _, w := range work {
        go func(w Work) {
            limit <- 1  // Acquire
            w.Do()
            <-limit     // Release
        }(w)
    }
    select{}
}
```

### Channel Closing

The closing of a channel happens before a receive that returns a zero value because the channel is closed.

**Example:**
```go
var c = make(chan int, 10)
var a string

func f() {
    a = "hello, world"
    close(c)
}

func main() {
    go f()
    <-c  // Receives zero value after close
    print(a)  // Guaranteed to print "hello, world"
}
```

### Locks (sync.Mutex and sync.RWMutex)

**For any `sync.Mutex` or `sync.RWMutex` variable `l`:**
- Call `n` of `l.Unlock()` happens before call `m` of `l.Lock()` returns (where `n < m`)

**Example:**
```go
var l sync.Mutex
var a string

func f() {
    a = "hello, world"
    l.Unlock()  // Call 0
}

func main() {
    l.Lock()    // Implicit call 0
    go f()
    l.Lock()    // Call 1 (waits for call 0 to Unlock)
    print(a)    // Guaranteed to print "hello, world"
}
```

**RWMutex:**
- Any call to `l.RLock()` returns after call `n` to `l.Unlock()`
- The corresponding call to `l.RUnlock()` happens before call `n+1` to `l.Lock()`

### Once

The `sync.Once` type provides a safe mechanism for initialization in the presence of multiple goroutines.

**Guarantee:**
A single call to `once.Do(f)` for a particular `f` happens before any `once.Do(f)` call returns.

**Example:**
```go
var a string
var once sync.Once

func setup() {
    a = "hello, world"
}

func doprint() {
    once.Do(setup)
    print(a)  // Guaranteed to print "hello, world"
}

func main() {
    go doprint()
    go doprint()
}
```

The first call to `once.Do(setup)` in either goroutine will execute `setup` once. Both calls to `doprint` will observe the write to `a`.

### Atomic Values (sync/atomic)

All operations in the `sync/atomic` package execute in a sequentially consistent order.

**Example:**
```go
var a int32 = 0
var wg sync.WaitGroup

func increment() {
    atomic.AddInt32(&a, 1)
    wg.Done()
}

func main() {
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go increment()
    }
    wg.Wait()
    fmt.Println(atomic.LoadInt32(&a))  // Guaranteed: 100
}
```

**Atomic vs mutex:**
- Atomics: best for simple counters and flags
- Mutexes: better for complex critical sections

### Finalizers

The call to `runtime.SetFinalizer(x, f)` happens before the finalizer call `f(x)`.

However, finalizers run unpredictably and should not be relied upon for program correctness.

## Common Mistakes

### Double-Checked Locking

**Wrong:**
```go
var done bool
var msg string

func setup() {
    msg = "hello, world"
    done = true  // Write
}

func main() {
    go setup()
    for !done {  // Read
    }
    print(msg)  // NOT guaranteed to observe "hello, world"
}
```

Even if `done` becomes true, there's no guarantee that the write to `msg` is visible.

**Correct:**
```go
var done = make(chan bool)
var msg string

func setup() {
    msg = "hello, world"
    done <- true
}

func main() {
    go setup()
    <-done
    print(msg)  // Guaranteed
}
```

### Busy Waiting

**Wrong:**
```go
var a, b int

func f() {
    a = 1
    b = 1
}

func g() {
    for b == 0 {  // Spin
    }
    print(a)  // NOT guaranteed to observe a=1
}

func main() {
    go f()
    g()
}
```

The compiler might optimize the loop to `for { }` since `b` is never modified in `g()`.

**Correct:**
```go
var a int
var done = make(chan bool)

func f() {
    a = 1
    done <- true
}

func g() {
    <-done
    print(a)  // Guaranteed
}

func main() {
    go f()
    g()
}
```

## Compiler Restrictions

To preserve memory model guarantees, compilers are restricted from:

1. **Introducing writes** not present in the original program
2. **Allowing single reads** to observe multiple values
3. **Allowing single writes** to write multiple values

**Example of prohibited optimization:**
```go
*p = 1
if cond {
    *p = 2
}

// Compiler CANNOT optimize to:
*p = 2
if !cond {
    *p = 1
}
```

This would introduce a write to `*p` on a code path where it didn't previously exist, potentially creating data races.

## Summary

**Best practices:**
1. Use channels for communication and synchronization
2. Use mutexes for protecting critical sections
3. Use atomics for simple shared counters/flags
4. Use `sync.Once` for one-time initialization
5. Avoid clever lock-free algorithms—they're error-prone

**Remember:**
If you need to reason carefully about memory ordering to understand your program's behavior, you're being too clever. Use explicit synchronization instead.
