# Golang Study...
updating... 💤

## Overview

- [x] **goredis** — A minimalist in-memory key-value store inspired by Redis, featuring dynamic hash table management and time-based key expiration.  
- [ ] **--** — _To be documented... 


---

## goredis Project

`goredis` is an educational implementation of a Redis-like in-memory data structure server, written in Go. It includes the following core features:

-  **Dynamic Dictionary with Rehashing**  
  Implements a dictionary (`Dict`) using dual hash tables to support **incremental rehashing**, reducing latency spikes during resizing.

-  **Collision Resolution via Linked Lists**  
  Each hash bucket handles collisions using separate chaining with linked list entries (`Entry`), ensuring efficient key-value management.

-  **Time-Based Key Expiration**  
  An auxiliary `expire` dictionary stores millisecond-precision expiration timestamps. Expired keys are periodically purged via a sampling strategy.

-  **Reference Counting for Memory Safety**  
  The core object (`Gobj`) utilizes reference counting to manage memory lifecycle, ensuring objects are properly retained and released.

-  **Active Expiration via `ServerCron`**  
  A scheduled routine performs randomized expiration checks and deletes stale keys from both `data` and `expire` dictionaries.

This module serves as a foundational component for building more complex systems, with emphasis on clarity, extensibility, and performance-aware design.

---

## quick Started

```bash
cd /goredis
sh init.sh


