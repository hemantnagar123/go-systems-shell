# POSIX-inspired Shell in Go

A custom interactive command-line shell interpreter implemented from scratch using only the Go standard library—built completely without relying on `readline` or existing third-party shell frameworks.

This project explores low-level systems programming, focusing on raw terminal I/O processing, custom asynchronous job management, stream routing pipelines, and manual terminal escape sequence parsing.

## 📺 Demo

> [!TIP]
> *Replace the placeholder lines below with your own recorded image/GIF links to show your shell in action!*


*Context-aware tab completion processing file structures.*


*Arrow-key history cycling and asynchronous background execution tracking.*

---

## 🛠️ Supported Features

* **Builtin Commands:** Implements custom internal handlers for `echo`, `pwd`, `cd`, `type`, `history`, `jobs`, and `complete`.
* **Job Control Workflows:** Supports asynchronous background process fork execution (`&`), real-time job registry listing, and task completion state reporting.
* **Pipeline Streams:** Coordinates multi-stage process chains (`external | external`, `builtin | external`, or `external | builtin`) via synchronous execution blocks.
* **Stream Redirection:** Full structural support for:
* Standard output overwrites (`>` or `1>`) and appends (`>>` or `1>>`).
* Standard error overwrites (`2>`) and appends (`2>>`).


* **Interactive Terminal Features:** Custom tab-completion tracking, programmable completion scripting hooks (`complete -C`), command session history tracking, and interactive up-arrow history navigation buffers.

---

## 🧠 The Technical Deep Dive: Canonical vs. Raw Mode

The fundamental challenge of this project was breaking away from basic line-by-line inputs to implement a real-time character interception loop.

```
+-----------------------------------------------------------------------------------+
|  CANONICAL MODE (Default OS)                                                     |
|  Keystrokes -> [OS Line Discipline Buffer] -> (Waits for \n) -> Passed to Go     |
+-----------------------------------------------------------------------------------+
                                         VS
+-----------------------------------------------------------------------------------+
|  RAW MODE (Custom Terminal Engine)                                                |
|  Keystrokes -> Passed Instantly Byte-by-Byte to os.Stdin.Read -> Manual Processing|
+-----------------------------------------------------------------------------------+

```

### The Canonical Constraints

Operating systems defaults to running terminals in **Canonical (Cooked) Mode**. The terminal driver buffers characters inside an internal line-discipline layer, handling deletions internally and waiting until the user presses **Enter** before sending anything to the application. This makes instant features like live tab-completions, single-character checking, or history scrolling structurally impossible.

### Transitioning to Raw Mode

To capture input interactively, the terminal descriptor is shifted into **Raw Mode** using `term.MakeRaw`. This turns off default terminal echo formatting and character preprocessing, feeding raw keystrokes directly into a continuous `os.Stdin.Read` byte interpretation loop.

Bypassing the OS layer solved the interactivity constraints but required manual handling of basic terminal events:

#### 1. The Ctrl+D (EOF) Trap

In canonical mode, pressing `Ctrl+D` causes the terminal driver to emit an empty stream status indicating `io.EOF`. In raw mode, that helper layer vanishes. Instead, `Ctrl+D` triggers a literal, raw ASCII control byte: **Value `4` (End of Transmission)**.

* **The Solution:** The character processing `switch` statement explicitly intercepts byte value `4`. If received while the input array buffer length is exactly zero, it prints a clean line break and forwards a manual `io.EOF` signal up to the caller to smoothly trigger the shell's logging and teardown routines.

#### 2. Re-Engineering Interactive Sequences

With high-level string reading disabled, standard text manipulation behaviors must be handled manually. Backspace keys emit raw byte `127` or `\b`. Arrow keys issue multi-byte ANSI escape sequences starting with a boundary escape character (`27`), followed by a bracket symbol (`[`) and directional flags (`A` for Up, `B` for Down).

* **The Solution:** We constructed custom array index manipulators. When backspace is pressed, the shell drops the last rune from the input array, enters a carriage return (`\r`), outputs the dynamic folder prompt path, rewrites the truncated string, and appends the ANSI terminal sequence `\033[K` to clear out trailing phantom characters from the screen display.

---

## 🛠️ Key Architectural Solutions

### 1. Synchronous Process Polling and Reaping

To prevent background jobs (`&`) from turning into dead zombie processes when they finish executing, the main execution engine needs to regularly track task life cycles. At the start of every instruction prompt loop, the shell runs a tracking method that pings active process IDs using a non-destructive signal check: `p.Signal(syscall.Signal(0))`. If the kernel signals that the process ID has dropped off the table, the state is updated to `Done` and the task is safely cleaned from memory.

### 2. Thread-Safe Job Registry

Because background execution tracking occurs across asynchronous, concurrent Go routines, modifying or viewing background tasks exposes the global state array to race conditions if a user checks running processes while a thread finishes. To ensure absolute data structure integrity, all updates, deletes, and list reads targeting the active background process slice are completely wrapped using a strict mutual exclusion lock (`sync.Mutex`).

### 3. Path Reconstruction Desynchronization

When parsing target completions that include explicit path references (such as `cd ./` or `../`), path-stripping utilities often remove prefix segments to read folder indexes, which can throw off character tracking on the input line. The autocomplete engine solves this by splitting text input to extract directory strings, scanning the targeted system location, and manually re-weaving the typed path prefixes back into the completion arrays before updating the screen.

---

## 💻 Getting Started

### Prerequisites

* Go 1.21 or higher installed on your local system.

### Build and Run

1. Clone this repository onto your machine:
```bash
git clone https://github.com/hemantnagar123/go-systems-shell.git
cd go-systems-shell

```


2. Compile the binary framework:
```bash
go build -o my_shell.exe .

```


3. Boot up your new custom interactive environment:
```bash
./my_shell.exe

```
