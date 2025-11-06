# Go Common Mistakes

This document addresses frequent programming errors in Go, with emphasis on patterns that commonly trip up developers.

## Loop Variable Scoping (Pre-Go 1.22)

**Note:** Go 1.22+ automatically creates iteration-scoped variables, making these workarounds unnecessary in modern code. However, understanding this pattern is important for working with older codebases.

### Problem: References to Loop Iterator Variables

When taking the address of a loop variable, all references point to the same memory location. The variable itself persists throughout iterations, changing its value each time.

**Problematic code:**
```go
var out []*int
for i := 0; i < 3; i++ {
    out = append(out, &i)
}
fmt.Println("values:", *out[0], *out[1], *out[2])
fmt.Println("addresses:", out[0], out[1], out[2])

// Output:
// values: 3 3 3
// addresses: 0x... 0x... 0x... (all same address!)
```

**Why this happens:**
- The loop variable `i` is allocated once, before the loop begins
- Each iteration updates the same memory location
- Taking `&i` captures the address of that single location
- After the loop, all pointers reference the final value (3)

**Solution: Create iteration-scoped copy**
```go
var out []*int
for i := 0; i < 3; i++ {
    i := i  // Creates new variable scoped to loop body
    out = append(out, &i)
}
fmt.Println("values:", *out[0], *out[1], *out[2])
// Output: values: 0 1 2
```

The line `i := i` creates a new variable with the same name, shadowing the loop variable within the loop body block. Each iteration gets its own distinct variable.

### Problem: Goroutines and Loop Variables

Closures launched as goroutines capture references to loop variables, not their values. Execution typically occurs after loop completion, accessing the final value.

**Problematic code:**
```go
values := []string{"a", "b", "c"}

for _, val := range values {
    go func() {
        fmt.Println(val)  // Captures reference to 'val'
    }()
}

time.Sleep(time.Second)

// Common output: c c c
// (all goroutines see the final value)
```

**Why this happens:**
- The goroutine closure captures a reference to `val`
- `val` is updated on each iteration
- Goroutines may not execute until after the loop completes
- By then, `val` contains the last value from the iteration

**Solution 1: Pass as parameter**
```go
for _, val := range values {
    go func(val string) {  // Parameter shadows loop variable
        fmt.Println(val)
    }(val)  // Value evaluated at call time
}

// Output: a b c (in some order)
```

The function parameter `val` shadows the loop variable, and the argument `val` is evaluated at the time of the function call (not when the goroutine executes).

**Solution 2: Create iteration-scoped variable**
```go
for _, val := range values {
    val := val  // Create new variable for this iteration
    go func() {
        fmt.Println(val)
    }()
}

// Output: a b c (in some order)
```

**Solution 3 (Go 1.22+): No workaround needed**
```go
// In Go 1.22+, this works correctly without modification
for _, val := range values {
    go func() {
        fmt.Println(val)  // Each iteration has its own 'val'
    }()
}

// Output: a b c (in some order)
```

## Nil Interfaces

A nil pointer stored in an interface is not a nil interface.

**Problematic code:**
```go
func returnsError() error {
    var p *MyError = nil
    if bad() {
        p = ErrBad
    }
    return p  // WRONG! Returns non-nil error interface
}

func main() {
    err := returnsError()
    if err != nil {
        fmt.Println("error occurred")  // Prints even when no error!
    }
}
```

**Why this happens:**
- An interface value consists of (type, value) pair
- When returning `p`, you're creating `error` interface containing `(*MyError, nil)`
- The interface is non-nil because it has a type, even though the value is nil
- `err != nil` is true because the interface itself is not nil

**Solution: Return explicit nil**
```go
func returnsError() error {
    var p *MyError = nil
    if bad() {
        p = ErrBad
        return p  // OK: returning actual error
    }
    return nil  // Return interface-level nil
}
```

**Alternative: Return error type directly**
```go
func returnsError() error {
    if bad() {
        return ErrBad
    }
    return nil
}
```

**Checking for typed nil:**
```go
if err != nil && reflect.ValueOf(err).IsNil() {
    // Interface is non-nil but contains nil pointer
}
```

## Map Concurrent Access

Maps are not thread-safe. Concurrent reads are safe, but any writes require synchronization.

**Problematic code:**
```go
m := make(map[string]int)

// Multiple goroutines writing
go func() { m["key1"] = 1 }()
go func() { m["key2"] = 2 }()  // RACE! Can crash
```

**Solution 1: Use sync.Mutex**
```go
var (
    m  = make(map[string]int)
    mu sync.RWMutex
)

func set(key string, val int) {
    mu.Lock()
    m[key] = val
    mu.Unlock()
}

func get(key string) int {
    mu.RLock()
    defer mu.RUnlock()
    return m[key]
}
```

**Solution 2: Use sync.Map for specific patterns**
```go
var m sync.Map

m.Store("key", value)
val, ok := m.Load("key")
m.Delete("key")
```

Use `sync.Map` when:
- Entry written once but read many times
- Multiple goroutines read, write, and overwrite disjoint key sets

Use regular map + mutex when:
- Access patterns don't match above
- You need range operations

## Slice Append Gotchas

Appending to a slice may or may not allocate a new underlying array.

**Problematic code:**
```go
func modify(s []int) {
    s[0] = 99     // Modifies original
    s = append(s, 100)  // May or may not allocate
    s[1] = 88     // May or may not affect original
}

original := []int{1, 2, 3}
modify(original)
fmt.Println(original)  // Could be [99, 2, 3] or [99, 88, 3]
```

**Why this happens:**
- Slices are passed by value, but reference underlying array
- If capacity sufficient, append modifies existing array
- If capacity exceeded, append allocates new array
- The slice in `modify` gets updated pointer, but `original` doesn't

**Solution: Return modified slice**
```go
func modify(s []int) []int {
    s[0] = 99
    s = append(s, 100)
    s[1] = 88
    return s
}

original := []int{1, 2, 3}
original = modify(original)
```

**Or use pointer to slice:**
```go
func modify(s *[]int) {
    (*s)[0] = 99
    *s = append(*s, 100)
    (*s)[1] = 88
}

original := []int{1, 2, 3}
modify(&original)
```

## Range Copies Values

Range iterates over copies of values, not references.

**Problematic code:**
```go
type Person struct {
    Name string
    Age  int
}

people := []Person{
    {"Alice", 30},
    {"Bob", 25},
}

for _, p := range people {
    p.Age++  // Modifies copy, not original!
}

fmt.Println(people[0].Age)  // Still 30
```

**Solution 1: Use index**
```go
for i := range people {
    people[i].Age++
}
```

**Solution 2: Use pointer slice**
```go
people := []*Person{
    {"Alice", 30},
    {"Bob", 25},
}

for _, p := range people {
    p.Age++  // Modifies through pointer
}
```

## Defer in Loops

Deferred functions execute when the surrounding function returns, not when loop iteration ends.

**Problematic code:**
```go
func processFiles(files []string) error {
    for _, filename := range files {
        f, err := os.Open(filename)
        if err != nil {
            return err
        }
        defer f.Close()  // Won't close until function returns!

        process(f)
    }
    return nil  // All files close here (may run out of file descriptors)
}
```

**Solution 1: Use helper function**
```go
func processFiles(files []string) error {
    for _, filename := range files {
        if err := processFile(filename); err != nil {
            return err
        }
    }
    return nil
}

func processFile(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close()  // Closes when processFile returns

    return process(f)
}
```

**Solution 2: Explicit close**
```go
func processFiles(files []string) error {
    for _, filename := range files {
        f, err := os.Open(filename)
        if err != nil {
            return err
        }

        err = process(f)
        f.Close()  // Close immediately

        if err != nil {
            return err
        }
    }
    return nil
}
```

## Short Variable Declaration Gotcha

`:=` can both declare new variables and assign to existing ones, but requires at least one new variable.

**Problematic code:**
```go
func example() error {
    err := errors.New("outer")

    if condition {
        result, err := doSomething()  // NEW err in inner scope!
        if err != nil {
            return err
        }
        use(result)
    }

    // 'err' here still holds "outer" error
    return err  // Wrong error!
}
```

**Solution: Use regular assignment when appropriate**
```go
func example() error {
    var err error

    if condition {
        var result SomeType
        result, err = doSomething()  // Assigns to outer err
        if err != nil {
            return err
        }
        use(result)
    }

    return err  // Correct
}
```

## Type Assertions Without Check

Type assertions without the two-value form panic on failure.

**Problematic code:**
```go
func process(val interface{}) {
    str := val.(string)  // Panics if val is not string!
    fmt.Println(strings.ToUpper(str))
}
```

**Solution: Use comma-ok idiom**
```go
func process(val interface{}) {
    str, ok := val.(string)
    if !ok {
        log.Printf("expected string, got %T", val)
        return
    }
    fmt.Println(strings.ToUpper(str))
}
```

**Or use type switch:**
```go
func process(val interface{}) {
    switch v := val.(type) {
    case string:
        fmt.Println(strings.ToUpper(v))
    case int:
        fmt.Println(v * 2)
    default:
        log.Printf("unexpected type %T", val)
    }
}
```

## Closing Channels

Only the sender should close a channel. Sending on a closed channel panics.

**Problematic code:**
```go
ch := make(chan int)

// Goroutine 1
go func() {
    ch <- 1
    close(ch)
}()

// Goroutine 2
go func() {
    ch <- 2  // Might panic if chan already closed!
}()
```

**Solution: Coordinate with sync.WaitGroup**
```go
ch := make(chan int)
var wg sync.WaitGroup

wg.Add(2)
go func() {
    defer wg.Done()
    ch <- 1
}()

go func() {
    defer wg.Done()
    ch <- 2
}()

go func() {
    wg.Wait()
    close(ch)  // Close after all senders done
}()

for val := range ch {
    fmt.Println(val)
}
```

## Method Value vs Method Expression

Method values capture the receiver; method expressions don't.

**Behavior:**
```go
type Counter struct {
    count int
}

func (c *Counter) Increment() {
    c.count++
}

// Method value - captures receiver
c := &Counter{}
f := c.Increment  // f is bound to this specific 'c'
f()               // Increments c.count

// Method expression - receiver passed as argument
c2 := &Counter{}
f2 := (*Counter).Increment  // f2 requires receiver argument
f2(c2)                      // Must pass receiver explicitly
```

**Gotcha with goroutines:**
```go
counters := []*Counter{{0}, {0}, {0}}

// Wrong - all goroutines use last counter
var f func()
for _, c := range counters {
    f = c.Increment  // f keeps getting reassigned
}
go f()  // Increments counters[2] only

// Correct - each goroutine captures its own method value
for _, c := range counters {
    go c.Increment()  // Each call creates method value with correct receiver
}
```

## JSON Marshaling Private Fields

JSON marshaling only encodes exported (capitalized) fields.

**Problematic code:**
```go
type User struct {
    name  string  // Won't be marshaled!
    email string  // Won't be marshaled!
}

u := User{"Alice", "alice@example.com"}
data, _ := json.Marshal(u)
fmt.Println(string(data))  // Output: {}
```

**Solution: Export fields**
```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

**For private storage with JSON:**
```go
type User struct {
    name  string
    email string
}

// Custom marshaler
func (u User) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }{
        Name:  u.name,
        Email: u.email,
    })
}
```
