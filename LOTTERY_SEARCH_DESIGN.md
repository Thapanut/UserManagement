# Lottery Search System - Architectural Design Proposal

## 1. Executive Summary

This document proposes a high-performance, real-world distributed system designed to search, distribute, and reserve **10,000,000 (10 million)** 6-digit lottery tickets (ranging from `000000` to `999999` across series/batches, or a general pool of 10M unique ticket IDs).

The system supports arbitrary 6-character wildcard queries (e.g., `****23`, `1****5`, `123***`, `*5*5*5`) and guarantees that **the same ticket is never returned or allocated to multiple concurrent users simultaneously**, achieving sub-millisecond to low-millisecond search latencies under high concurrency.

---

## 2. System Requirements & Challenge Analysis

### 2.1 Scale & Constraints
- **Ticket Dataset**: 10,000,000 tickets. Each ticket has a 6-digit number (`000000` – `999999`), ticket UID, batch/round ID, price, status (`AVAILABLE`, `RESERVED`, `SOLD`), and timestamps.
- **Number vs. UID cardinality**: a 6-digit number space only has 1,000,000 possible values, so with 10M tickets each number is duplicated across ~10 distinct physical tickets (different batches/draws/series). All bitmap indexing and reservation logic below operates on the **ticket UID** (a dense integer ID from 0 to 9,999,999 used purely for bit-position addressing), not on the printed 6-digit number itself. The 6-digit number is only used to build the positional index; multiple UIDs can map to the same number.
- **Query Flexibility**: 6-character patterns containing digits `[0-9]` and wildcards `*` at any position (e.g., `*8888*`, `12****`, `****59`).
- **Core Invariant (Zero-Duplicate Allocation)**: Under bursty concurrent traffic (e.g., during high-demand lottery release hours), if 100 users search `****88` at the same second, each user MUST receive distinct ticket results or available quotas without collision or double-reservation.
- **Latency Target**: P99 read + reserve latency < 25ms.

---

## 3. High-Level Architecture

```
                       +-------------------------+
                       |   Clients / Web & App   |
                       +------------+------------+
                                    |
                                    v
                       +-------------------------+
                       |     API Gateway /       |
                       |  Cloud Load Balancer    |
                       +------------+------------+
                                    |
                                    v
                       +-------------------------+
                       |  Lottery Search Service | (Stateless Go instances)
                       +-----+-------------+-----+
                             |             |
           +-----------------+             +-----------------+
           |                                                 |
           v                                                 v
+-------------------------------+             +-------------------------------+
|  Redis (Primary + Replicas)   |             |     Primary Persistent DB     |
|  Sharded by draw_id via       |             | PostgreSQL 16 (Partitioned)  |
|  hash-tagged keys, per-draw   |             |  - Permanent storage          |
|  Lua Script                   |             |  - Transactional audit log    |
|  - Positional Inverted Index  |             |  - Trigram/B-Tree Indexes     |
|  - Real-time Bitmap/Set       |             |                               |
|  - Atomic Reservation Lease   |             |                               |
+-------------------------------+             +-------------------------------+
                                                             ^
                                                             | Change Data Capture
                                              +--------------+--------------+
                                              | Debezium / Kafka Async Sync |
                                              +-----------------------------+
```

### Key Architectural Layers:
1. **API Gateway & Rate Limiter**: Distributes incoming traffic, protects against bot scrapers.
2. **Lottery Search & Allocation Service**: Stateless Golang services providing search and lease checkout endpoints.
3. **In-Memory Query & Allocation Engine (Redis Cluster)**: Handles pattern matching and atomic ticket leases in sub-milliseconds.
4. **Relational Database (PostgreSQL 16)**: Serves as the durable source of truth, managing finalized transactions, financial audits, and payment states.

---

## 4. Data Structures & Indexing Strategy

Searching 10M records with random wildcards (like `*5*5*5` or `**12**`) causes full-table scans in traditional databases unless an appropriate indexing scheme is selected.

### 4.1 Positional Inverted Index (Radix Digit Sets)

Since lottery tickets are strictly 6 characters long and each position $i \in \{0, 1, 2, 3, 4, 5\}$ only has 10 possible values (`'0'` through `'9'`), the search space across all positions is exceptionally compact:
$$\text{Total Positional Keys} = 6 \text{ positions} \times 10 \text{ digits} = 60 \text{ index buckets}.$$

In Redis, we maintain 60 Roaring Bitmaps or Redis Sets, all scoped to a specific `draw_id` and wrapped in a **hash tag** so Redis Cluster guarantees they live on the same slot/node (required because `BITOP`/`SINTER` fail across slots):
- `pos:{draw123}:0:digit:0` ... `pos:{draw123}:0:digit:9`
- `pos:{draw123}:1:digit:0` ... `pos:{draw123}:1:digit:9`
- ...
- `pos:{draw123}:5:digit:0` ... `pos:{draw123}:5:digit:9`

Along with an **Availability Bitmap / Set**:
- `tickets:{draw123}:available` (10M bits $\approx$ **1.25 Megabytes** of memory!).

Because every key for a given draw shares the `{draw123}` hash tag, Redis Cluster routes them all to the same node, so cross-key bitwise ops stay valid; horizontal scale is then achieved by spreading *different draws* across *different nodes/slots* rather than splitting one draw's index across nodes (see §7.2).

#### How Matching Works:
For a query such as `1****5` on draw `draw123`:
1. Position 0 must be `1` $\rightarrow$ set `pos:{draw123}:0:digit:1`
2. Position 5 must be `5` $\rightarrow$ set `pos:{draw123}:5:digit:5`
3. Status must be available $\rightarrow$ `tickets:{draw123}:available`
4. The candidate ticket IDs are resolved via **Bitwise AND (`BITOP AND`)** or **Set Intersection (`SINTER`)** — valid because all three keys share the `{draw123}` hash tag and live on the same node:
   $$\text{Result} = \text{pos:0:1} \cap \text{pos:5:5} \cap \text{tickets:available}$$

#### Memory Footprint:
- 10,000,000 tickets represented as a bitmap = $10,000,000 \text{ bits} = 1.19 \text{ MB}$.
- 60 positional bitmaps $\times 1.2 \text{ MB} \approx 72 \text{ MB}$ total!
- The entire 10-million ticket index fits into **less than 100 MB of RAM**, allowing Redis to execute bitwise intersections in microseconds.

---

## 5. Concurrency & Non-Duplicate Result Distribution Strategy

The requirement states:
> *"The same search pattern should not return the same ticket to multiple users at the same time. Propose a distribution mechanism so matching tickets are assigned without duplicate simultaneous selection."*

If User A and User B query `****23` simultaneously, simply returning the top 5 matches to both will result in both seeing and trying to purchase the exact same tickets.

### 5.1 Atomic Two-Phase Lease Allocation (Redis Lua Script)

Instead of a passive read-only query, searching operates under an **Atomic Search-and-Reserve (Lease)** model:

```mermaid
sequenceDiagram
    autonumber
    actor UserA as User A
    actor UserB as User B
    participant API as Search API
    participant Redis as Redis (Lua Script)
    participant Reaper as Reconciliation Worker
    participant PG as PostgreSQL

    UserA->>API: Search "****23" (Limit 5)
    UserB->>API: Search "****23" (Limit 5)

    API->>Redis: EVALSHA SearchAndHold("****23", UserA_ID, Limit=5, TTL=60s)
    API->>Redis: EVALSHA SearchAndHold("****23", UserB_ID, Limit=5, TTL=60s)

    Note over Redis: Redis executes Lua atomically (Single-Threaded)
    Note over Redis: Each lease is also written into leases:{draw_id}:expirations (ZSET)
    Redis-->>API: Returns [Ticket 000023, 000123, 000223, 000323, 000423] to User A
    Note over Redis: Tickets removed from available bitmap & added to User A's hold set
    Redis-->>API: Returns [Ticket 000523, 000623, 000723, 000823, 000923] to User B

    API-->>UserA: 5 Tickets (Held for 60s)
    API-->>UserB: 5 Different Tickets (Held for 60s)

    loop Every 1s
        Reaper->>Redis: ZRANGEBYSCORE leases:{draw_id}:expirations -inf now
        Reaper->>Redis: Release expired leases (restore bit, delete lease, ZREMRANGEBYSCORE)
    end

    UserA->>API: POST /api/v1/tickets/release (cancels before TTL)
    API->>Redis: Release UserA's held tickets immediately (same Lua release path)
```

#### Detailed Lua Algorithm:
1. **Intersection**: Performs `BITOP AND` across fixed-position keys and `tickets:available` into a temporary key.
2. **Fair Selection (Randomized Offset)**: Rather than always taking the first $N$ set bits (which would deterministically favor low-numbered tickets and hand the same "front of the queue" tickets to whoever searches first), the script picks a **random starting bit offset** into the intersected bitmap and walks forward (wrapping around) using `BITPOS`, collecting the first $N$ positive bits it encounters from that offset. This keeps selection O(N) while avoiding low-ID bias.
3. **Atomic State Transition**:
   - Clears the bits in `tickets:available` (immediate removal from subsequent searches).
   - Sets a Redis Hash `lease:<ticket_id>` $\rightarrow$ `{user_id, expires_at}` with a 60-second TTL.
   - Adds the ticket IDs to `user:<user_id>:held_tickets`.
4. **Expiration / Auto-Rollback (Reliable ZSET Reaper)**:
   - Relying solely on **Redis Keyspace Notifications** (`EXPIRE` + pub/sub) is **not production-safe**: keyspace events are delivered **at-most-once** over Redis pub/sub, and a dropped connection, network blip, or worker restart at the wrong moment means the "key expired" event is simply lost. There is no re-delivery, no queue, no acknowledgment — the ticket's bit never gets cleared and it is **permanently stuck in a reserved state**, silently shrinking available inventory over time.
   - Instead, every lease additionally registers itself in a **Redis Sorted Set**, `leases:{draw_id}:expirations`, where the **member** is the `ticket_id` and the **score** is the `expiration_unix_timestamp`. This ZSET is durable Redis state (not a fire-and-forget event), so nothing is lost even if a consumer is briefly offline.
   - A lightweight **background reconciliation worker** polls this ZSET roughly every **1 second**:
     1. `ZRANGEBYSCORE leases:{draw_id}:expirations -inf <now>` to fetch all leases that have expired.
     2. For each expired `ticket_id`, atomically (via a small Lua script) sets the bit back to `1` in `tickets:{draw_id}:available`, deletes the corresponding `lease:<ticket_id>` hash, and removes it from `user:<user_id>:held_tickets`.
     3. `ZREMRANGEBYSCORE leases:{draw_id}:expirations -inf <now>` to clear the processed entries from the ZSET.
   - Because the ZSET is polled and re-scanned rather than depending on a single pub/sub delivery, a missed poll cycle (worker crash, deploy, GC pause) simply means the next cycle picks up the same still-present ZSET entries — **no lease can silently vanish without being reconciled**. This guarantees **100% lease rollback with zero inventory leakage**, at the cost of up to ~1 second of extra latency before an expired ticket becomes searchable again (an acceptable trade-off against the correctness this buys).
   - If the user confirms payment before expiry, PostgreSQL records the sale, the ticket is removed from `leases:{draw_id}:expirations` (so the reaper never processes it), and Redis permanently flags the ticket as `SOLD`.

#### Explicit Release API (Inventory Starvation Mitigation)
Waiting out the full 60-second TTL on every abandoned search is wasteful under load: if a large share of users search, glance at results, and navigate away or cancel without buying, those tickets sit needlessly reserved-but-unsold for up to a minute each, starving other concurrent searchers of inventory that is, in practice, free.

To mitigate this, the API exposes an explicit release endpoint:

```
POST /api/v1/tickets/release
{
  "user_id": "UserA_ID",
  "ticket_ids": ["000023", "000123", "000223"]
}
```

- Called automatically by the client on navigation-away, explicit "cancel"/"clear results", or search-results dismissal (e.g. via a `beforeunload`/route-change hook), and can also be called explicitly if the user removes individual tickets from their held selection before checkout.
- The handler runs the **same atomic release logic** as the reaper (clear the bit in `tickets:{draw_id}:available`, delete the `lease:<ticket_id>` hash, remove from `user:<user_id>:held_tickets`, and remove the entry from `leases:{draw_id}:expirations`) via a shared Lua script, so both paths — timeout-based and explicit — converge on identical, race-free state transitions.
- This turns the 60-second TTL into a **worst-case backstop** rather than the common case, so inventory returns to the pool in milliseconds for the (typically large) share of users who abandon their search intentionally, rather than accidentally.

#### Redis Durability & Cold-Start Rebuild
Redis here is a **derived, rebuildable index**, not the source of truth — PostgreSQL is. To avoid data loss on a Redis crash/restart:
- Enable Redis **AOF persistence** (`appendfsync everysec`) plus RDB snapshots for fast restart, so most state survives a normal restart.
- On a cold start (empty Redis) or detected corruption, a **bootstrap job** rebuilds all 60 positional bitmaps and the `tickets:available` bitmap directly from PostgreSQL (`SELECT id, number FROM tickets WHERE status = 'AVAILABLE'`), setting bits accordingly. For 10M rows this is a single sequential scan and completes in seconds.
- Active leases (`lease:<ticket_id>`) are also mirrored as `RESERVED` rows in PostgreSQL when created, so an in-flight reservation is not lost even if Redis restarts mid-lease; the bootstrap job excludes rows still marked `RESERVED` and not yet expired.

---

## 6. Recommended Database & Storage Technology

| Storage Layer | Technology Choice | Justification & Production Suitability |
| --- | --- | --- |
| **Primary In-Memory & Indexing** | **Redis Enterprise / AWS ElastiCache, sharded per-draw via hash tags** | - **Ultra-low latency**: Bitwise operations run in sub-millisecond time.<br>- **Single-threaded atomicity**: Lua scripts guarantee 100% race-condition-free allocation without complex distributed lock managers.<br>- **Compact memory**: 10M tickets index requires < 100MB RAM.<br>- **Cluster-safe sharding**: each draw's 60+1 keys share a hash tag so `BITOP`/Lua stay single-node valid; different draws are distributed across nodes/slots for horizontal scale. |
| **Primary Persistent Datastore** | **PostgreSQL 16 (Declarative Partitioning)** | - **ACID Compliance**: Crucial for financial settlements and inventory integrity.<br>- **Range Partitioning**: Partitioned by ticket batch / draw date (`PARTITION BY RANGE (draw_date)`).<br>- **Optimistic Locking**: Uses `xmin` or `version` columns for final checkout defense-in-depth (`WHERE status = 'RESERVED' AND id = ...`). |
| **Search / Analytics (Auxiliary)** | **Elasticsearch / OpenSearch** | - If users need complex full-text metadata searches (e.g., ticket series name, lottery store location, dealer info). |

---

## 7. Performance Analysis & Trade-offs

### 7.1 Algorithmic Complexity

| Operation | Strategy | Time Complexity | Latency (Estimated) |
| --- | --- | --- | --- |
| **Exact match** (`123456`) | Direct Hash lookup | $O(1)$ | $< 0.5 \text{ ms}$ |
| **Suffix match** (`****23`) | Bitwise AND (2 keys + available) | $O(\frac{N}{64}) \approx 156,000 \text{ ops}$ | $1.2 \text{ ms}$ |
| **Prefix match** (`123***`) | Bitwise AND (3 keys + available) | $O(\frac{N}{64})$ | $1.0 \text{ ms}$ |
| **Arbitrary wildcard** (`*2*4*6`) | Bitwise AND (3 keys + available) | $O(\frac{N}{64})$ | $1.5 \text{ ms}$ |
| **Ticket Reservation** | In-script `BITTEST & CLEAR` | $O(K)$ where $K = \text{batch size}$ | $< 0.5 \text{ ms}$ |

### 7.2 Scalability & Bottlenecks

1. **Memory Scalability**:
   - 10M tickets = 1.19MB per bitset.
   - 100M tickets = 11.9MB per bitset.
   - Even scaling to 100 million tickets, the memory footprint remains under 1 GB of RAM!
2. **CPU Bound on Bitwise AND**:
   - Modern CPUs process 64-bit to 256-bit SIMD registers simultaneously. Performing `AND` across 1.2MB takes approximately 0.2 milliseconds in Redis C code.
3. **Partitioning Strategy**:
   - If ticket pools expand into multiple lottery categories or draw dates, datasets are sharded by `draw_id`. Each draw operates its own independent 60-bucket index (all keys hash-tagged as `{draw_id}`, per §4.1), enabling horizontal sharding across Redis Cluster nodes without ever requiring a cross-node `BITOP`.
4. **Match-Count Independence**:
   - Because the search cost is a fixed bitwise AND over the full bitmap, latency is essentially **constant regardless of how many tickets match** — a pattern matching 2 tickets and one matching 200,000 tickets both cost roughly the same ~1-2ms, which is a meaningful advantage over index-scan approaches whose cost grows with result-set size.

---

## 8. Summary of Design Benefits

1. **Zero Overbooking / Collision**: Guaranteed by Redis Lua atomic script isolation.
2. **Blazing Fast**: Average search-and-allocate time of 1–3 ms on 10M records.
3. **Cost-Effective**: Requires negligible infrastructure costs (< $50/month cloud memory for the indexing tier).
4. **Resilient**: Graceful lease timeouts ensure tickets are never locked indefinitely if a user abandons their shopping cart.
