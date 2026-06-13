# POSIX-inspired Shell in Go

A custom interactive command-line shell interpreter implemented from scratch using only the Go standard library—built completely without relying on `readline` or existing third-party shell frameworks.

This project explores low-level systems programming, focusing on raw terminal I/O processing, custom asynchronous job management, stream routing pipelines, and manual terminal escape sequence parsing.

## 📺 Demo

> [!TIP]







https://github.com/user-attachments/assets/29ac1163-f10a-47a6-88b2-2d7207c2eee5



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

Shifting the terminal into **Raw Mode** via `term.MakeRaw` feeds keystrokes directly into `os.Stdin.Read` byte-by-byte. This bypassed automatic OS line-buffering but required handling standard terminal inputs manually:

* **The Ctrl+D Trap:** Raw mode captures `Ctrl+D` as the literal ASCII control byte `4` instead of a traditional `io.EOF`. The loop explicitly intercepts this byte on an empty line to forward a manual exit signal.
* **Manual Line Editing & Escape Sequences:** Backspace keys emit raw byte `127`, and arrow keys send multi-byte ANSI escape sequences (like `\x1b[A` for up-arrow). The shell handles these by updating an internal rune slice and redrawing the terminal line using carriage returns (`\r`) and clearing codes (`\033[K`).
---

## 🛠️ Key Architectural Solutions

* **Zombie Process Reaping:** Pings active background processes at the start of every prompt loop using `p.Signal(syscall.Signal(0))`. This tracks task lifecycles and cleanly removes completed jobs from memory.
  
* **Thread-Safe Job Registry:** Protects the shared background jobs array with a `sync.Mutex` to prevent concurrent read/write data races between asynchronous Go routines.
  
* **Path Reconstruction:** Tracks typed relative path prefixes (like `./` or `../`) during tab completion, manually re-weaving them into matching outputs to prevent input-line desynchronization.

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
