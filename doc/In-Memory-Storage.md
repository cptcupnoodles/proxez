# In-Memory Ledger (IML) with WAL

An ultra-low latency, file-backed storage architecture designed for edge networking. It pairs the raw execution speed of thread-safe system memory with the crash-resilient permanence of sequential disk logging.

## How to Implement It

Implementing this system requires two distinct storage layers operating in a strict, sequential mutation pipeline:
