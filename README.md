# termshare

Share your terminal with anyone on the same network — instantly, no config required.

- Zero setup: one command to host, one to join
- Join code authentication — only who you approve gets in
- Both host and client share the same shell session
- Auto-discovery on LAN via mDNS

---

## Security & Trust Warning

**Sharing a terminal gives someone full access to your machine.** This is more powerful than screen sharing — the other person can run any command, read any file, and modify anything your user account can touch.

Before using termshare, understand the following:

- **Only share with people you fully trust.** Once approved, the client has the same access as you do in that shell.
- **Traffic is not encrypted.** Anyone on the same network can intercept keystrokes and output using tools like Wireshark. Do not share sensitive credentials or private data over a termshare session.
- **The join code is not a strong secret.** It is 6 characters and is transmitted over plaintext TCP. It prevents accidental connections, not determined attackers on your network.
- **mDNS advertises your presence.** While hosting, your session is visible to everyone on the LAN via `termshare list`. There is no stealth mode.
- **This tool is not a replacement for SSH.** SSH is encrypted, key-authenticated, and battle-tested. Use SSH if security matters.

**Recommended use cases:**
- Helping a teammate debug something quickly on a trusted private network
- Teaching or live demos in a classroom or workshop
- Your own machines on your own home network

**Avoid using termshare:**
- On public WiFi (cafes, airports, conferences)
- With people you don't fully trust
- For sessions involving passwords, API keys, or private data

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

Starts a shell session and prints a join code. You'll first be asked to choose a session mode:

```
Session mode:
❯ Read + Write  (client can type)
  Read only     (client can only watch)
```

Then the session starts:

```
termshare — hosting session
Mode      : read-only
Join code : A3F9K2
Port      : 4321

Waiting for a connection... (Ctrl+C to stop)
```

When someone tries to join, you'll see a prompt:

```
[?] Connection request from 192.168.1.8 — approve? [y/N]:
```

Type `y` to accept.

- **Read + Write** — both host and client share the same shell. Either can type.
- **Read only** — client sees everything but their input is ignored. Useful for demos, teaching, or walkthroughs.

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

The simplest way — just run with no args and pick from a dropdown:

```sh
termshare join
```

Or even shorter:

```sh
termshare
```

This scans the LAN, shows available sessions, and prompts for the join code.

For direct connect:

```sh
termshare join 192.168.1.5:4321 -c A3F9K2
```

Once approved by the host, you'll see:

```
Connected.            ← read+write mode
Connected (read-only). ← read-only mode
```

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

- [x] Read-only viewer mode
- [ ] Encryption (TLS)
- [ ] mDNS fallback to port scan for networks that block multicast
- [ ] Multi-user sessions
- [ ] Internet relay (WebRTC)
