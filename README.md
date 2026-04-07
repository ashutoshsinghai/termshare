# termshare

Share your terminal with anyone on the same network — instantly, no config required.

- Zero setup: one command to host, one to join
- Join code authentication — only who you approve gets in
- Both host and client share the same shell session
- Auto-discovery on LAN via mDNS

---

## Installation

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/ashutoshsinghai/termshare/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
iwr https://raw.githubusercontent.com/ashutoshsinghai/termshare/main/scripts/install.ps1 | iex
```

### Windows (CMD)

```cmd
powershell -ExecutionPolicy Bypass -Command "iwr https://raw.githubusercontent.com/ashutoshsinghai/termshare/main/scripts/install.ps1 | iex"
```

> After installation on Windows, restart your terminal for PATH changes to take effect.

### Manual

Download the binary for your platform from the [latest release](https://github.com/ashutoshsinghai/termshare/releases/latest) and place it in your PATH.

---

## Usage

### Host a session

```sh
termshare host
```

Starts a shell session and prints a join code. Anyone on the same network can join using that code.

```
termshare — hosting session
Join code : A3F9K2
Port      : 4321

Waiting for a connection... (Ctrl+C to stop)
```

When someone tries to join, you'll see a prompt:

```
[?] Connection request from 192.168.1.8 — approve? [y/N]:
```

Type `y` to accept. Both of you will share the same terminal session.

---

### Discover sessions

```sh
termshare list
```

Scans the local network and lists active sessions:

```
Available sessions:
  ashutosh-mac.local.  →  192.168.1.5:4321
```

---

### Join a session

```sh
termshare join <host:port> -c <join-code>
```

Example:

```sh
termshare join 192.168.1.5:4321 -c A3F9K2
```

Once approved by the host, you'll share their terminal in real time.

> Press `Ctrl+\` to disconnect without closing the remote session.

---

## How it works

```
Host                          Client
 │                              │
 │  termshare host              │  termshare join <ip> -c <code>
 │  → starts PTY (shell)        │
 │  → TCP server on :4321       │  → connects via TCP
 │  → mDNS advertisement        │
 │                              │
 │  ← join code verified        │  → sends join code
 │  ← host approves [y/N]       │  ← waiting for approval...
 │                              │
 │  PTY output → stdout + TCP → │  → displayed on client terminal
 │  host stdin → PTY            │
 │                    client input → TCP → PTY
```

- Protocol: binary framed messages `[1 byte type][4 bytes length][N bytes payload]`
- Discovery: mDNS (`_termshare._tcp`)
- Terminal resize (SIGWINCH) is relayed so programs like `vim` and `htop` render correctly

---

## Building from source

Requires Go 1.22+.

```sh
git clone https://github.com/ashutoshsinghai/termshare.git
cd termshare
go build -o termshare .
```

---

## Roadmap

- [ ] mDNS fallback to port scan for networks that block multicast
- [ ] Read-only viewer mode
- [ ] Multi-user sessions
- [ ] Internet relay (WebRTC)
