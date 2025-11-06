# Effective Go - Complete Reference Guide

## Introduction

Go is a new language designed for modern programming. While different from traditional languages, idiomatic Go code borrows successfully from ancestors like C and Pascal while introducing unique features for concurrent programming and other modern needs.

## Formatting

### Use gofmt

All Go code should be formatted with `gofmt` (or `goimports`, which also fixes imports). This removes all debates about formatting and ensures consistency across the ecosystem.

**Key points:**
- Tabs for indentation (gofmt default)
- No line length limit (but be reasonable)
- Less indentation than C/Java (no need to comment closing braces)
- Parentheses usage is reduced

### Commentary

Go provides C-style `/* */` block comments and C++-style `//` line comments. Line comments are the norm.

**Doc comments:**
- Every package should have a package comment (before package clause)
- Multi-file packages: only one file needs package comment
- Start with "Package name" and give a brief description
- Every exported name should have a doc comment
- First sentence should be a summary starting with the name

Example:
```go
// Package sort provides primitives for sorting slices and user-defined collections.
package sort

// Fprint formats using the default formats for its operands and writes to w.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Fprint(w io.Writer, a ...interface{}) (n int, err error) {
```

## Names

Names are as important in Go as in any other language. They even have semantic effect: the visibility of a name outside a package is determined by whether its first character is uppercase.

### Package names

- When importing, package name becomes accessor for contents
- Should be short, concise, evocative
- By convention: lowercase, single-word names
- No need for underscores or mixedCaps
- Package name is used as prefix, so don't repeat it in exported names

Examples:
- `bufio.Reader` not `bufio.BufReader` (users see `bufio.Reader`)
- `once.Do` not `once.DoOrWaitUntilDone`

### Getters

Go doesn't provide automatic support for getters and setters. Nothing wrong with providing them yourself, but it's neither idiomatic nor necessary to put `Get` in the getter's name.

```go
owner := obj.Owner()       // Good
if owner != user {
    obj.SetOwner(user)
}
```

Not:
```go
owner := obj.GetOwner()    // Not idiomatic
```

### Interface names

By convention, one-method interfaces are named by the method name plus an -er suffix: `Reader`, `Writer`, `Formatter`, `CloseNotifier`, etc.

### MixedCaps

Go convention is to use `MixedCaps` or `mixedCaps` rather than underscores for multi-word names.

## Semicolons

Like C, Go's formal grammar uses semicolons to terminate statements. Unlike C, those semicolons do not appear in the source. The lexer uses a simple rule to insert semicolons automatically as it scans:

If the newline comes after a token that could end a statement, insert a semicolon.

**Consequence:** You cannot put the opening brace of a control structure on the next line:

```go
// Wrong - will not compile
if i < f()
{
    g()
}

// Correct
if i < f() {
    g()
}
```

## Control structures

### If

Simple if statements:
```go
if x > 0 {
    return y
}
```

Can include initialization:
```go
if err := file.Chmod(0664); err != nil {
    log.Print(err)
    return err
}
```

### Redeclaration and reassignment

In a `:=` declaration, a variable v may appear even if it has already been declared, provided:
- This declaration is in the same scope as the existing declaration of v
- The corresponding value in the initialization is assignable to v, and
- There is at least one other variable in the declaration that is being created anew

This allows you to use a single `err` value across multiple operations:
```go
f, err := os.Open(name)
if err != nil {
    return err
}
d, err := f.Stat()  // err redeclared here
if err != nil {
    f.Close()
    return err
}
```

### For

Go has only one looping construct: `for`. It can be used in three forms:

```go
// Like a C for
for init; condition; post { }

// Like a C while
for condition { }

// Like a C for(;;)
for { }
```

Range form for strings, slices, maps:
```go
for key, value := range oldMap {
    newMap[key] = value
}

// If you only need first item (key or index):
for key := range m {
    if key.expired() {
        delete(m, key)
    }
}

// If you only need second item:
for _, value := range array {
    sum += value
}
```

**String iteration:**
```go
for pos, char := range "日本\x80語" {  // \x80 is an illegal UTF-8 encoding
    fmt.Printf("character %#U starts at byte position %d\n", char, pos)
}
```

Output:
```
character U+65E5 '日' starts at byte position 0
character U+672C '本' starts at byte position 3
character U+FFFD '�' starts at byte position 6
character U+8A9E '語' starts at byte position 7
```

### Switch

Go's switch is more flexible than C's:
- Expressions need not be constants or integers
- Cases are evaluated top to bottom until a match
- No automatic fall through (break not needed)
- Can have no expression (switch true)

```go
func unhex(c byte) byte {
    switch {
    case '0' <= c && c <= '9':
        return c - '0'
    case 'a' <= c && c <= 'f':
        return c - 'a' + 10
    case 'A' <= c && c <= 'F':
        return c - 'A' + 10
    }
    return 0
}
```

**Type switch:**
```go
var t interface{}
t = functionOfSomeType()
switch t := t.(type) {
case bool:
    fmt.Printf("boolean %t\n", t)
case int:
    fmt.Printf("integer %d\n", t)
case *bool:
    fmt.Printf("pointer to boolean %t\n", *t)
default:
    fmt.Printf("unexpected type %T\n", t)
}
```

## Functions

### Multiple return values

One of Go's unusual features is that functions and methods can return multiple values. This form can improve several clumsy idioms in C programs:
- In-band error returns (such as -1 for EOF)
- Passing a pointer to a return value

```go
func (file *File) Write(b []byte) (n int, err error)
```

### Named result parameters

The return or result "parameters" of a Go function can be given names and used as regular variables.

```go
func ReadFull(r Reader, buf []byte) (n int, err error) {
    for len(buf) > 0 && err == nil {
        var nr int
        nr, err = r.Read(buf)
        n += nr
        buf = buf[nr:]
    }
    return
}
```

### Defer

Go's `defer` statement schedules a function call to be run immediately before the function executing the defer returns.

```go
func Contents(filename string) (string, error) {
    f, err := os.Open(filename)
    if err != nil {
        return "", err
    }
    defer f.Close()  // f.Close will run when we're finished

    var result []byte
    buf := make([]byte, 100)
    for {
        n, err := f.Read(buf[0:])
        result = append(result, buf[0:n]...)
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", err
        }
    }
    return string(result), nil
}
```

**Defer advantages:**
- Guarantees cleanup regardless of return path
- Deferred functions execute in LIFO order
- Function arguments are evaluated when defer executes

**Useful patterns:**
```go
// Trace function entry/exit
func trace(s string) string {
    fmt.Println("entering:", s)
    return s
}

func un(s string) {
    fmt.Println("leaving:", s)
}

func a() {
    defer un(trace("a"))
    fmt.Println("in a")
}

// Output:
// entering: a
// in a
// leaving: a
```

## Data

### Allocation with new

`new(T)` allocates zeroed storage for a new item of type T and returns its address (a value of type `*T`).

```go
type SyncedBuffer struct {
    lock    sync.Mutex
    buffer  bytes.Buffer
}

p := new(SyncedBuffer)  // type *SyncedBuffer
var v SyncedBuffer      // type  SyncedBuffer
```

### Constructors and composite literals

Sometimes the zero value isn't good enough and an initializing constructor is necessary.

```go
func NewFile(fd int, name string) *File {
    if fd < 0 {
        return nil
    }
    f := File{fd, name, nil, 0}
    return &f
}
```

Can be simplified to:
```go
return &File{fd, name, nil, 0}
```

Or with field labels:
```go
return &File{fd: fd, name: name}
```

**Arrays, slices, maps composite literals:**
```go
a := [...]string   {Enone: "no error", Eio: "Eio", Einval: "invalid argument"}
s := []string      {Enone: "no error", Eio: "Eio", Einval: "invalid argument"}
m := map[int]string{Enone: "no error", Eio: "Eio", Einval: "invalid argument"}
```

### Allocation with make

`make(T, args)` creates slices, maps, and channels only, and returns an initialized (not zeroed) value of type T (not `*T`).

```go
var p *[]int = new([]int)       // allocates slice structure; *p == nil
var v  []int = make([]int, 100) // v refers to a new array of 100 ints

// Unnecessarily complex:
var p *[]int = new([]int)
*p = make([]int, 100, 100)

// Idiomatic:
v := make([]int, 100)
```

### Arrays

- Arrays are values (not pointers)
- Passing an array to a function makes a copy
- Size is part of the type: `[10]int` and `[20]int` are distinct

```go
func Sum(a *[3]float64) (sum float64) {
    for _, v := range *a {
        sum += v
    }
    return
}

array := [...]float64{7.0, 8.5, 9.1}
x := Sum(&array)
```

### Slices

Slices wrap arrays to give a more general, powerful, and convenient interface to sequences of data.

```go
func (f *File) Read(buf []byte) (n int, err error)
```

**Slice internals:**
- Hold references to underlying array
- If you pass a slice, changes are visible to caller
- Length can change (up to capacity)

```go
func Append(slice, data []byte) []byte {
    l := len(slice)
    if l + len(data) > cap(slice) {  // reallocate
        newSlice := make([]byte, (l+len(data))*2)
        copy(newSlice, slice)
        slice = newSlice
    }
    slice = slice[0:l+len(data)]
    copy(slice[l:], data)
    return slice
}
```

**Built-in append:**
```go
x := []int{1,2,3}
x = append(x, 4, 5, 6)
```

### Two-dimensional slices

```go
type Transform [3][3]float64  // A 3x3 array, really an array of arrays
type LinesOfText [][]byte     // A slice of byte slices

// Allocate the top-level slice:
text := make([][]byte, 0, 100)  // Initial capacity 100

// Allocate slices one at a time:
for i := 0; i < 10; i++ {
    text = append(text, make([]byte, 100))
}
```

### Maps

Maps are convenient, powerful built-in data structure associating values of one type (key) with values of another type (value).

```go
var timeZone = map[string]int{
    "UTC":  0*60*60,
    "EST": -5*60*60,
    "CST": -6*60*60,
}
```

**Testing for presence:**
```go
offset, ok := timeZone["UTC"]
if !ok {
    log.Println("not found")
}
```

**Idiom to test without caring about value:**
```go
_, present := timeZone["UTC"]
```

**Delete a key:**
```go
delete(timeZone, "PDT")
```

### Printing

```go
fmt.Printf("Hello %d\n", 23)
fmt.Fprintf(os.Stdout, "Hello %d\n", 23)
fmt.Sprintf("Hello %d", 23)
```

**Formats:**
- `%v` - default format
- `%+v` - includes field names for structs
- `%#v` - Go syntax representation
- `%T` - type of value
- `%t` - boolean
- `%d` - integer
- `%x`, `%o`, `%b` - hex, octal, binary
- `%f`, `%g`, `%e` - floats
- `%s` - string
- `%q` - quoted string
- `%p` - pointer

### Append

```go
func append(slice []T, elements ...T) []T
```

The built-in append must return a result slice because the underlying array may change.

```go
x := []int{1,2,3}
y := []int{4,5,6}
x = append(x, y...)  // ... required to pass slice as variadic argument
```

## Initialization

### Constants

Constants are created at compile time, even when defined as locals, and can only be numbers, characters (runes), strings, or booleans.

```go
type ByteSize float64

const (
    _           = iota  // ignore first value
    KB ByteSize = 1 << (10 * iota)
    MB
    GB
    TB
    PB
    EB
    ZB
    YB
)
```

### The init function

Each source file can define its own `init()` function to set up state.

```go
func init() {
    if user == "" {
        log.Fatal("$USER not set")
    }
    if home == "" {
        home = "/home/" + user
    }
    if gopath == "" {
        gopath = home + "/go"
    }
}
```

## Methods

### Pointers vs. Values

Methods can be defined for any named type (except a pointer or an interface).

The rule about pointers vs. values for receivers is that value methods can be invoked on pointers and values, but pointer methods can only be invoked on pointers.

```go
type ByteSlice []byte

func (slice ByteSlice) Append(data []byte) []byte {
    // Body as above
}

func (p *ByteSlice) Append(data []byte) {
    slice := *p
    // Body as above
    *p = slice
}
```

**Pointer receivers:**
- Can modify the receiver
- More efficient for large structs
- Consistent when some methods need pointers

## Interfaces and other types

### Interfaces

Interfaces provide a way to specify the behavior of an object: if something can do this, then it can be used here.

```go
type Sequence []int

// Methods required by sort.Interface
func (s Sequence) Len() int {
    return len(s)
}
func (s Sequence) Less(i, j int) bool {
    return s[i] < s[j]
}
func (s Sequence) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

// Method for printing
func (s Sequence) String() string {
    s = s.Copy()
    sort.Sort(s)
    return fmt.Sprint([]int(s))
}
```

### Conversions

The `String` method above makes use of a conversion technique:
```go
func (s Sequence) String() string {
    s = s.Copy()
    sort.Sort(s)
    return fmt.Sprint([]int(s))  // Convert to []int to use default printing
}
```

### Interface conversions and type assertions

Type switches are a form of conversion: they take an interface and, for each case in the switch, convert it to the type of that case.

```go
type Stringer interface {
    String() string
}

var value interface{}  // Value from caller
switch str := value.(type) {
case string:
    return str
case Stringer:
    return str.String()
}
```

### Generality

If a type exists only to implement an interface and will never have exported methods beyond that interface, there is no need to export the type itself.

```go
// Only export the interface
type Handler interface {
    Handle(w Writer, r *Request)
}

// Unexported concrete type
type handlerFunc func(Writer, *Request)

func (f handlerFunc) Handle(w Writer, r *Request) {
    f(w, r)
}
```

### Interfaces and methods

Since almost anything can have methods attached, almost anything can satisfy an interface. One example is `http.Handler`:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

## The blank identifier

### The blank identifier in multiple assignment

```go
if _, err := os.Stat(path); os.IsNotExist(err) {
    fmt.Printf("%s does not exist\n", path)
}
```

### Unused imports and variables

It is an error to import a package or declare a variable without using it. Use blank identifier:

```go
package main

import (
    "fmt"
    "io"
    "log"
    "os"
)

var _ = fmt.Printf  // For debugging; delete when done
var _ io.Reader     // For debugging; delete when done

func main() {
    fd, err := os.Open("test.go")
    if err != nil {
        log.Fatal(err)
    }
    _ = fd  // TODO: use fd
}
```

### Import for side effect

To import a package only for its side effects:

```go
import _ "net/http/pprof"
```

### Interface checks

A value need not declare explicitly that it implements an interface. Sometimes it's useful to guarantee at compile time:

```go
var _ json.Marshaler = (*RawMessage)(nil)
```

## Embedding

Go allows embedding types within structs and interfaces to "borrow" pieces of an implementation.

```go
type ReadWriter struct {
    *Reader  // *bufio.Reader
    *Writer  // *bufio.Writer
}
```

This is embedding, not subclassing. The embedded types' methods become methods of the outer type, but when invoked the receiver is the inner type, not the outer one.

```go
type Job struct {
    Command string
    *log.Logger
}

job := &Job{command, log.New(os.Stderr, "Job: ", log.Ldate)}
job.Println("starting now...")  // Calls (*job.Logger).Println
```

### Conflicts

If overlapping names occur, the more deeply nested name is hidden. This gives control if there's a collision.

## Concurrency

### Share by communicating

> Do not communicate by sharing memory; instead, share memory by communicating.

One way to think about this model is to consider a typical single-threaded program running on one CPU. It has no need for synchronization primitives. Now run another such instance; it too needs no synchronization. Now let those two communicate; if the communication is the synchronizer, there's still no need for other synchronization.

### Goroutines

They're called goroutines because the existing terms—threads, coroutines, processes, and so on—convey inaccurate connotations. A goroutine has a simple model: it is a function executing concurrently with other goroutines in the same address space.

```go
go list.Sort()  // run list.Sort concurrently; don't wait
```

Goroutines are lightweight, costing little more than the allocation of stack space. Stacks start small and grow by allocating/freeing heap storage as needed.

### Channels

Like maps, channels are allocated with `make`, and the resulting value acts as a reference to an underlying data structure.

```go
ci := make(chan int)            // unbuffered channel of integers
cj := make(chan int, 0)         // unbuffered channel of integers
cs := make(chan *os.File, 100)  // buffered channel of pointers
```

**Unbuffered channels combine communication with synchronization, guaranteeing that two calculations (goroutines) are in a known state.**

```go
c := make(chan int)  // Allocate channel

// Start sort in goroutine
go func() {
    list.Sort()
    c <- 1  // Send signal; value doesn't matter
}()

doSomethingForAWhile()
<-c   // Wait for sort to finish
```

**Buffered channels can be used as semaphores:**
```go
var sem = make(chan int, MaxOutstanding)

func handle(r *Request) {
    sem <- 1    // Wait for active queue to drain
    process(r)  // May take a long time
    <-sem       // Done; enable next request
}

func Serve(queue chan *Request) {
    for {
        req := <-queue
        go handle(req)
    }
}
```

### Channels of channels

Channels are first-class values and can be allocated and passed around like any other. A common use is to implement safe, parallel demultiplexing.

```go
type Request struct {
    args        []int
    f           func([]int) int
    resultChan  chan int
}

func sum(a []int) (s int) {
    for _, v := range a {
        s += v
    }
    return
}

request := &Request{[]int{3, 4, 5}, sum, make(chan int)}
clientRequests <- request   // Send request
fmt.Printf("answer: %d\n", <-request.resultChan)
```

### Parallelization

```go
type Vector []float64

func (v Vector) DoAll(u Vector) {
    for i := range v {
        v[i] += u[i]
    }
}
```

Parallel version:
```go
const numCPU = 4  // or runtime.NumCPU()

func (v Vector) DoAll(u Vector) {
    c := make(chan int, numCPU)
    for i := 0; i < numCPU; i++ {
        go v.DoSome(i*len(v)/numCPU, (i+1)*len(v)/numCPU, u, c)
    }
    // Drain channel
    for i := 0; i < numCPU; i++ {
        <-c
    }
}

func (v Vector) DoSome(i, j int, u Vector, c chan int) {
    for ; i < j; i++ {
        v[i] += u[i]
    }
    c <- 1
}
```

### A leaky buffer

```go
var freeList = make(chan *Buffer, 100)
var serverChan = make(chan *Buffer)

func client() {
    for {
        var b *Buffer
        select {
        case b = <-freeList:
            // Got one; nothing more to do
        default:
            // None free, allocate new
            b = new(Buffer)
        }
        load(b)              // Read next message from network
        serverChan <- b      // Send to server
    }
}

func server() {
    for {
        b := <-serverChan    // Wait for work
        process(b)
        select {
        case freeList <- b:
            // Buffer on free list; nothing more
        default:
            // Free list full, discard buffer
        }
    }
}
```

## Errors

### Error type

```go
type error interface {
    Error() string
}
```

### Custom errors

```go
type PathError struct {
    Op   string
    Path string
    Err  error
}

func (e *PathError) Error() string {
    return e.Op + " " + e.Path + ": " + e.Err.Error()
}
```

### Panic

The usual way to report an error is to return an `error` value. The canonical Read method demonstrates:

```go
func (f *File) Read(buf []byte) (n int, err error)
```

`panic` is a built-in function that stops ordinary flow of control. When a function calls `panic`, execution stops, deferred functions run, and the function returns to its caller. That continues up the stack until all functions in the current goroutine have returned, at which point the program crashes.

### Recover

When `panic` is called, it immediately stops execution of the current function and begins unwinding the goroutine's stack, running any deferred functions. If that unwinding reaches the top of the goroutine's stack, the program dies. However, it is possible to use `recover` to regain control and resume normal execution.

```go
func server(workChan <-chan *Work) {
    for work := range workChan {
        go safelyDo(work)
    }
}

func safelyDo(work *Work) {
    defer func() {
        if err := recover(); err != nil {
            log.Println("work failed:", err)
        }
    }()
    do(work)
}
```

## A web server

```go
package main

import (
    "flag"
    "html/template"
    "log"
    "net/http"
)

var addr = flag.String("addr", ":1718", "http service address")
var templ = template.Must(template.New("qr").Parse(templateStr))

func main() {
    flag.Parse()
    http.Handle("/", http.HandlerFunc(QR))
    err := http.ListenAndServe(*addr, nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}

func QR(w http.ResponseWriter, req *http.Request) {
    templ.Execute(w, req.FormValue("s"))
}

const templateStr = `
<html>
<head>
<title>QR Link Generator</title>
</head>
<body>
{{if .}}
<img src="http://chart.apis.google.com/chart?chs=300x300&cht=qr&choe=UTF-8&chl={{.}}" />
<br>
{{.}}
<br>
<br>
{{end}}
<form action="/" name=f method="GET">
    <input maxLength=1024 size=70 name=s value="" title="Text to QR Encode">
    <input type=submit value="Show QR" name=qr>
</form>
</body>
</html>
`
```
