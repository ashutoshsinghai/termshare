# Product Requirements Document (PRD)

## Product Name

**termshare** (LAN MVP)

---

## 1. Overview

termshare is a CLI tool that allows developers to share and access terminal sessions across devices on the same local network (LAN) in real time.

The goal of the MVP is to provide a **zero-config, instant, peer-to-peer terminal sharing experience** for local environments.

---

## 2. Problem Statement

Developers frequently need to:

* Debug issues together
* Help teammates remotely
* Access another machine’s terminal

Existing tools are:

* Complex to set up (SSH configs, ports)
* Not optimized for quick collaboration
* Dependent on external services

There is a need for a **simple, instant, LAN-based solution**.

---

## 3. Goals

### Primary Goals

* Enable real-time terminal sharing over LAN
* Zero configuration required
* Discoverable sessions automatically
* Fast and reliable connection

### Secondary Goals

* Simple file transfer within session
* Clean and intuitive CLI UX

---

## 4. Non-Goals (MVP Scope)

* Internet-based connections
* Authentication and access control
* Multi-user sessions
* Encryption (optional later)
* Session persistence

---

## 5. Target Users

* Developers
* DevOps engineers
* Students learning programming
* Teams working on shared networks

---

## 6. Key Features

### 6.1 Host Session

Command:

```
termshare host
```

Behavior:

* Starts a shell session (bash/zsh)
* Opens TCP server on a port
* Advertises session via mDNS

---

### 6.2 Discover Sessions

Command:

```
termshare list
```

Behavior:

* Discovers active sessions on LAN
* Displays hostname and address

Example Output:

```
Available sessions:
- ashutosh-mac (192.168.1.5:4321)
- office-server (192.168.1.8:4321)
```

---

### 6.3 Join Session

Command:

```
termshare join <host>
```

Behavior:

* Connects to selected host
* Starts interactive terminal session

---

### 6.4 Interactive Mode (Optional UX Improvement)

Command:

```
termshare
```

Behavior:

* Lists available sessions
* Allows user to select one interactively

---

### 6.5 File Transfer (Basic)

Command (inside session):

```
/send file.txt
```

Behavior:

* Sends file over existing connection
* Saves on receiver side

---

## 7. User Flow

### Host Flow

1. User runs `termshare host`
2. Session starts
3. Session is advertised on LAN

### Client Flow

1. User runs `termshare list`
2. Sees available sessions
3. Runs `termshare join <host>`
4. Terminal session starts

---

## 8. Technical Architecture

### Components

#### 8.1 Discovery Layer

* mDNS (zeroconf)
* Broadcasts available sessions

#### 8.2 Networking Layer

* TCP server (host)
* TCP client (joiner)

#### 8.3 Terminal Layer

* PTY (pseudo-terminal)
* Captures stdin/stdout

#### 8.4 Protocol Layer

* Custom lightweight protocol

Message Types:

* INPUT
* OUTPUT
* FILE

---

## 9. Data Flow

1. Host starts PTY
2. Client connects via TCP
3. Client input → sent to host
4. Host output → streamed back

---

## 10. CLI Design

### Commands

* `termshare host`
* `termshare list`
* `termshare join <host>`
* `termshare`

---

## 11. Success Metrics

* Time to connect (< 2 seconds)
* Zero configuration success rate
* Session stability
* Developer adoption (qualitative)

---

## 12. Risks & Mitigations

### Risk: Network incompatibility

* Mitigation: Clear error messages

### Risk: Terminal sync issues

* Mitigation: Use PTY instead of manual execution

### Risk: Discovery failures

* Mitigation: Allow manual IP connection

---

## 13. Future Roadmap

### Phase 2

* Multi-user sessions
* Read-only mode
* File transfer improvements

### Phase 3

* Internet support (WebRTC)
* Relay fallback
* Encryption

---

## 14. MVP Timeline

### Day 1

* TCP communication

### Day 2

* PTY integration

### Day 3

* mDNS discovery
* CLI polish

---

## 15. Vision

termshare aims to become:

> "The simplest way to share and collaborate in a terminal—locally first, globally later."
