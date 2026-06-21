# Learn Go Through go-probe

> A hands-on tutorial for go-probe's author. You already have Python experience — this guide teaches Go by comparing it to what you already know, using every line of `main.go` as the example.

---

## Table of Contents
1. [Why Go feels different from Python](#1-why-go-feels-different-from-python)
2. [Packages and imports](#2-packages-and-imports)
3. [Types and structs](#3-types-and-structs)
4. [Functions and methods](#4-functions-and-methods)
5. [Pointers — the one tricky thing](#5-pointers--the-one-tricky-thing)
6. [Goroutines — Go's superpower](#6-goroutines--gos-superpower)
7. [WaitGroup — waiting for goroutines to finish](#7-waitgroup--waiting-for-goroutines-to-finish)
8. [Mutex — protecting shared data](#8-mutex--protecting-shared-data)
9. [HTTP client — making requests](#9-http-client--making-requests)
10. [HTTP server — serving requests](#10-http-server--serving-requests)
11. [Interfaces](#11-interfaces)
12. [Error handling](#12-error-handling)
13. [The `main` function and CLI flags](#13-the-main-function-and-cli-flags)
14. [Putting it all together — trace a single probe cycle](#14-putting-it-all-together--trace-a-single-probe-cycle)
15. [Exercises to deepen understanding](#15-exercises-to-deepen-understanding)

---

## 1. Why Go feels different from Python

| Feature | Python | Go |
|---|---|---|
| Typing | Dynamic — types checked at runtime | Static — types checked at compile time |
| Speed | Interpreted, slower | Compiled to machine code, fast |
| Concurrency | Threads + GIL (painful) | Goroutines — lightweight, built-in |
| Error handling | Exceptions (`try/except`) | Explicit return values (`error`) |
| Classes | `class Foo:` | `struct` + methods — no inheritance |
| Running code | `python main.py` | `go build && ./go-probe` or `go run main.go` |

The biggest mental shift: **Go is explicit**. Python hides a lot of complexity. Go makes you spell it out — which is why it's easier to read in a team.

---

## 2. Packages and imports

```go
package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "strings"
    "sync"
    "time"
)
```

**Python equivalent:**
```python
import json
import argparse
import time
import logging
import threading
```

**Key points:**
- Every `.go` file starts with `package <name>`. Files in the same folder share a package.
- `package main` is special — it's the entry point. Only one package in a project can be `main`.
- Go's standard library is huge. Everything we use in go-probe (`sync`, `net/http`, `time`, `flag`) is built in — **no pip install**.
- Unused imports are a **compile error** in Go. If you import something and don't use it, it won't build.

---

## 3. Types and structs

Python has classes. Go has **structs** — just a collection of named fields with types. No inheritance.

```go
// ProbeResult holds the outcome of a single health check against one target URL.
type ProbeResult struct {
    URL        string        `json:"url"`
    Up         bool          `json:"up"`
    StatusCode int           `json:"status_code"`
    Latency    time.Duration `json:"latency_ms"`
    CheckedAt  time.Time     `json:"checked_at"`
    Error      string        `json:"error,omitempty"`
}
```

**Python equivalent:**
```python
from dataclasses import dataclass
from datetime import datetime, timedelta

@dataclass
class ProbeResult:
    url: str
    up: bool
    status_code: int
    latency: timedelta
    checked_at: datetime
    error: str = ""
```

**Breaking down the Go syntax:**

`string`, `bool`, `int` — Go's basic types. Python has the same concepts but you don't have to declare them.

`time.Duration` — a type from the `time` package. Represents a span of time (e.g., 142ms). Like Python's `timedelta`.

`` `json:"url"` `` — the part in backticks is a **struct tag**. It tells the `encoding/json` package what name to use when serialising this field to JSON. `omitempty` means: skip this field in the JSON output if the value is empty/zero.

**Creating a struct value:**
```go
result := &ProbeResult{
    URL:       url,
    CheckedAt: start,
}
```

The `&` means "give me a pointer to this struct" — covered in section 5.

---

## 4. Functions and methods

**A regular function:**
```go
func NewProber(targets []string, timeout time.Duration) *Prober {
    return &Prober{
        targets: targets,
        timeout: timeout,
        results: make(map[string]*ProbeResult),
    }
}
```

**Python equivalent:**
```python
def new_prober(targets: list[str], timeout: timedelta) -> "Prober":
    return Prober(targets=targets, timeout=timeout, results={})
```

**Go function syntax:**
```
func FunctionName(param1 type1, param2 type2) returnType {
    ...
}
```
- Parameters: name first, then type (opposite of Python type hints)
- Return type goes at the end after the closing `)`
- Functions can return multiple values: `func divide(a, b int) (int, error)`

**A method (function attached to a struct):**
```go
func (p *Prober) probe(url string) {
    // p is like Python's self
}
```

**Python equivalent:**
```python
def probe(self, url: str):
    ...
```

In Go, `(p *Prober)` is the **receiver** — it's like `self` in Python, but you declare it explicitly with a name and type before the function name. The `*` means it's a pointer receiver (see section 5).

**Visibility — capital letters matter:**
```go
func (p *Prober) RunOnce() { ... }  // exported — callable from other packages
func (p *Prober) probe(url string)  // unexported — private to this package
```

In Python you use `_` prefix for private. In Go: capital = public, lowercase = private.

---

## 5. Pointers — the one tricky thing

A pointer is just the **memory address** of a value. You've seen `*Prober` and `&Prober{}` — here's what they mean:

```go
// Without pointer — a COPY of Prober is created
func (p Prober) probe(url string) { ... }

// With pointer — operates on the ORIGINAL Prober
func (p *Prober) probe(url string) { ... }
```

**Why it matters in go-probe:** `Prober` holds a `results` map. If we didn't use a pointer receiver, every method call would get a copy of `Prober`, and changes to `results` would be thrown away. With `*Prober`, all methods operate on the same object.

**The two pointer operators:**
```go
p := &Prober{}    // & = "give me the address of this"   → p is type *Prober
fmt.Println(*p)   // * = "give me the value at this address" → dereferences the pointer
```

**Python analogy:** In Python, objects are always passed by reference — you never think about this. In Go, simple values (int, bool, struct) are copied by default; you opt into reference semantics with `*`.

**Rule of thumb:** If a method modifies the struct's fields, use a pointer receiver `*`.

---

## 6. Goroutines — Go's superpower

A goroutine is a lightweight thread managed by Go. Starting one is a single keyword: `go`.

```go
// In RunOnce():
for _, target := range p.targets {
    wg.Add(1)
    go func(url string) {   // <-- "go" launches this as a goroutine
        defer wg.Done()
        p.probe(url)
    }(target)               // <-- immediately call the function with target as argument
}
```

**Python equivalent (threading):**
```python
import threading

threads = []
for target in self.targets:
    t = threading.Thread(target=self.probe, args=(target,))
    t.start()
    threads.append(t)
for t in threads:
    t.join()
```

**What's happening:**
- For each target URL, we start a goroutine that calls `p.probe(url)`
- All probes run **simultaneously** — if we have 10 targets, all 10 HTTP requests fire at the same time
- Go can run thousands of goroutines on a few OS threads — they're cheap

**Anonymous function (closure) syntax:**
```go
go func(url string) {
    // url is a local copy — safe to use in a goroutine
    p.probe(url)
}(target)  // this parenthesis calls the function immediately, passing target as url
```

Why not just `go p.probe(target)`? That would also work here. The anonymous function pattern is used when you need to capture variables safely across loop iterations.

---

## 7. WaitGroup — waiting for goroutines to finish

Goroutines are "fire and forget" — if `main` returns, they're all killed. We need a way to wait for all probes to finish before we declare the cycle done.

```go
var wg sync.WaitGroup   // a counter, starts at 0

for _, target := range p.targets {
    wg.Add(1)           // increment counter: "one more goroutine starting"
    go func(url string) {
        defer wg.Done() // decrement counter when this goroutine exits
        p.probe(url)
    }(target)
}

wg.Wait()               // block here until counter reaches 0
```

**Python equivalent:**
```python
# What WaitGroup does is exactly what Thread.join() loop does
for t in threads:
    t.join()
```

`defer wg.Done()` — `defer` means "run this when the current function returns, no matter what". It's like Python's `finally`. Always pair `wg.Add(1)` with a deferred `wg.Done()` so the counter always decrements even if the goroutine panics.

---

## 8. Mutex — protecting shared data

The `results` map is written by many goroutines (one per probe) and read by HTTP handler goroutines simultaneously. Without protection, this is a **data race** — undefined, buggy behaviour.

```go
type Prober struct {
    ...
    mu      sync.RWMutex          // the lock
    results map[string]*ProbeResult
}
```

**Writing (exclusive lock):**
```go
p.mu.Lock()                       // only ONE goroutine can hold this
p.results[url] = result
p.mu.Unlock()                     // release it
```

**Reading (shared lock):**
```go
p.mu.RLock()                      // MANY goroutines can read at once
defer p.mu.RUnlock()
// ... read from p.results ...
```

**Python equivalent:**
```python
import threading
lock = threading.RLock()

# Write
with lock:
    self.results[url] = result

# Read
with lock:
    return list(self.results.values())
```

**RWMutex vs Mutex:**
- `sync.Mutex` — only one goroutine at a time, whether reading or writing
- `sync.RWMutex` — multiple readers at once OR one writer (not both) — better for read-heavy workloads like go-probe where the HTTP /metrics endpoint reads constantly but writes are less frequent

---

## 9. HTTP client — making requests

```go
func (p *Prober) probe(url string) {
    client := &http.Client{Timeout: p.timeout}
    start := time.Now()

    result := &ProbeResult{
        URL:       url,
        CheckedAt: start,
    }

    resp, err := client.Get(url)   // returns (response, error)
    result.Latency = time.Since(start)

    if err != nil {
        result.Up = false
        result.Error = err.Error()
    } else {
        defer resp.Body.Close()           // always close the response body
        result.StatusCode = resp.StatusCode
        result.Up = resp.StatusCode >= 200 && resp.StatusCode < 400
    }

    p.mu.Lock()
    p.results[url] = result
    p.mu.Unlock()
}
```

**Python equivalent:**
```python
import requests

def probe(self, url: str):
    start = time.time()
    result = ProbeResult(url=url, checked_at=datetime.now())
    try:
        resp = requests.get(url, timeout=self.timeout.total_seconds())
        result.latency = timedelta(seconds=time.time() - start)
        result.status_code = resp.status_code
        result.up = 200 <= resp.status_code < 400
    except Exception as e:
        result.up = False
        result.error = str(e)
```

**Key Go patterns here:**

`resp, err := client.Get(url)` — Go returns errors as values, not exceptions. The convention is always `(result, error)`. You must check `err != nil` before using `result`.

`defer resp.Body.Close()` — HTTP responses must have their body closed or you leak a connection. `defer` ensures it happens when `probe()` returns.

`time.Now()` / `time.Since(start)` — measuring elapsed time. `time.Since(start)` returns a `time.Duration`.

---

## 10. HTTP server — serving requests

```go
http.HandleFunc("/metrics", prober.handleMetrics)
http.HandleFunc("/status", prober.handleStatus)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, `go-probe — endpoints: /metrics (Prometheus), /status (JSON)`)
})

log.Fatal(http.ListenAndServe(*addr, nil))
```

**Python equivalent (Flask):**
```python
@app.route("/metrics")
def handle_metrics():
    ...

@app.route("/status")
def handle_status():
    ...

app.run(host="0.0.0.0", port=8080)
```

**Handler signature — always the same:**
```go
func (p *Prober) handleStatus(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(...)
}
```

- `w http.ResponseWriter` — write the response to this (like `return` in Flask, but a writer)
- `r *http.Request` — the incoming request (method, URL, headers, body)
- `w.Header().Set(...)` — set response headers before writing the body
- `json.NewEncoder(w).Encode(data)` — write JSON directly to the response writer

---

## 11. Interfaces

Go interfaces are satisfied **implicitly** — you don't have to say "I implement this interface". If your type has the right methods, it qualifies.

`http.ResponseWriter` is an interface:
```go
type ResponseWriter interface {
    Header() Header
    Write([]byte) (int, error)
    WriteHeader(statusCode int)
}
```

Any type that has those three methods is an `http.ResponseWriter`. Our handlers accept it without knowing (or caring) what concrete type is underneath.

**Python analogy:** Duck typing — if it quacks like a duck, it's a duck. Go's interfaces formalise this at compile time.

---

## 12. Error handling

Go has no exceptions. Errors are just values returned from functions.

```go
if err := http.ListenAndServe(*addr, nil); err != nil {
    log.Fatal(err)
}
```

**Python equivalent:**
```python
try:
    server.serve_forever()
except Exception as e:
    logging.fatal(e)
    sys.exit(1)
```

**The pattern is always:**
```go
result, err := someFunction()
if err != nil {
    // handle the error — return it, log it, or panic
}
// safe to use result here
```

`log.Fatal(err)` — prints the error and calls `os.Exit(1)`. Use it when the error is unrecoverable.

---

## 13. The `main` function and CLI flags

```go
func main() {
    targetsFlag := flag.String("targets", "", "Comma-separated list of URLs to probe")
    interval    := flag.Duration("interval", 30*time.Second, "How often to re-probe")
    timeout     := flag.Duration("timeout", 5*time.Second, "HTTP request timeout per probe")
    addr        := flag.String("addr", ":8080", "Address to serve on")
    flag.Parse()

    if *targetsFlag == "" {
        log.Fatal("--targets is required")
    }
    ...
}
```

**Python equivalent:**
```python
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--targets", required=True)
parser.add_argument("--interval", default=30, type=int)
args = parser.parse_args()
```

**How `flag` works:**
- `flag.String("targets", "", "description")` registers a flag and returns a **pointer** to a string (`*string`)
- After `flag.Parse()`, that pointer points to whatever was passed on the command line (or the default)
- Access the value with `*targetsFlag` (dereference the pointer)

**Running:**
```bash
./go-probe --targets=https://example.com,https://google.com --interval=15s
```

---

## 14. Putting it all together — trace a single probe cycle

When you run `go-probe`, here's exactly what happens, step by step:

```
main() starts
│
├── Registers CLI flags, parses them
├── Creates a Prober:  prober := NewProber(targets, timeout)
│     └── Initialises the results map: make(map[string]*ProbeResult)
│
├── Calls prober.RunOnce()  ← blocking, waits for all probes
│     ├── For each target URL, starts a goroutine:
│     │     └── goroutine: probe("https://example.com")
│     │           ├── Creates http.Client with timeout
│     │           ├── Records start time
│     │           ├── Makes GET request → gets response or error
│     │           ├── Records latency = time.Since(start)
│     │           ├── Acquires write lock  (p.mu.Lock())
│     │           ├── Stores result in p.results["https://example.com"]
│     │           └── Releases write lock (p.mu.Unlock())
│     │
│     └── wg.Wait() — blocks until ALL goroutines call wg.Done()
│
├── Starts prober.RunLoop(interval) in a goroutine ← runs forever in background
│     └── Calls RunOnce() every 30 seconds
│
├── Registers HTTP handlers: /metrics, /status, /
│
└── http.ListenAndServe(":8080") — blocks forever, handling HTTP requests
      │
      ├── GET /metrics → handleMetrics()
      │     ├── Acquires read lock (p.mu.RLock())
      │     ├── Reads p.results
      │     ├── Releases read lock (p.mu.RUnlock())
      │     └── Writes Prometheus text format to response
      │
      └── GET /status → handleStatus()
            ├── Calls p.Results() which acquires RLock
            └── Writes JSON to response
```

---

## 15. Exercises to deepen understanding

Work through these in order — each one builds on the last. Try them on your local copy of `main.go`.

### Exercise 1 — Add a new field to ProbeResult
Add a `Redirects int` field to `ProbeResult` that counts how many HTTP redirects happened.
> Hint: `http.Client` follows redirects by default. Look at `CheckRedirect` in the `http.Client` struct.

### Exercise 2 — Add a `/healthz` endpoint
Add a new handler at `/healthz` that returns `{"status":"ok"}` with HTTP 200 — always, regardless of probe results. This is a standard Kubernetes liveness probe pattern.

### Exercise 3 — Add a probe count metric
Add a `TotalProbes int` counter to `Prober` that increments each time any URL is probed. Expose it in `/metrics` as `probe_total`.
> Watch out: `TotalProbes` is shared across goroutines — how do you protect it?

### Exercise 4 — Make timeout configurable per target
Right now all targets share one timeout. Change the data model so targets can be defined as `"https://example.com:10s"` — URL followed by a colon and a duration.
> Hint: `strings.SplitN(target, ":", 2)` and `time.ParseDuration()`

### Exercise 5 — Write a test
Create `main_test.go`. Write a test for `probe()` that starts a local HTTP test server, calls `probe()` on it, and asserts the result is `Up: true` with a non-zero latency.
> Hint: `net/http/httptest.NewServer()` creates a local test HTTP server

### Exercise 6 — Read from a JSON config file
Instead of `--targets=url1,url2`, support a `--config=targets.json` flag where the file looks like:
```json
[
  {"url": "https://example.com", "name": "Example"},
  {"url": "https://google.com",  "name": "Google"}
]
```
> Hint: `os.ReadFile()`, `json.Unmarshal()`

---

## Quick reference: Go vs Python

| Go | Python |
|---|---|
| `var x int = 5` | `x: int = 5` |
| `x := 5` | `x = 5` (inferred) |
| `[]string{"a", "b"}` | `["a", "b"]` |
| `map[string]int{"a": 1}` | `{"a": 1}` |
| `make(map[string]int)` | `{}` |
| `for i, v := range slice` | `for i, v in enumerate(list)` |
| `for k, v := range m` | `for k, v in dict.items()` |
| `if err != nil` | `if error is not None` |
| `fmt.Println("hello")` | `print("hello")` |
| `fmt.Sprintf("%s %d", s, n)` | `f"{s} {n}"` |
| `log.Printf("msg %v", val)` | `logging.info("msg %s", val)` |
| `strings.Split(s, ",")` | `s.split(",")` |
| `strings.Join(ss, ",")` | `",".join(ss)` |
