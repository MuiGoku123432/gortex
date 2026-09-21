# API Coverage -- Neo4j Go Driver

> Full coverage by default. Opt-outs are explicit, reasoned decisions. This matrix records the external Neo4j driver/API surface considered for the scoped snapshot projection.

| capability | decision | reason |
|---|---|---|
| named-profile resolution with environment-referenced credentials | INTEGRATE | |
| URI userinfo rejection and secret-safe errors | INTEGRATE | |
| Bolt connectivity verification | INTEGRATE | |
| explicit database selection | INTEGRATE | |
| Neo4j server version and edition inspection | INTEGRATE | |
| Neo4j 5.26 LTS compatibility enforcement | INTEGRATE | |
| node and relationship constraint inspection | INTEGRATE | |
| node and per-relationship-type constraint creation during apply | INTEGRATE | |
| non-mutating constraint and target census during dry-run | INTEGRATE | |
| parameterized Cypher values | INTEGRATE | |
| static sanitized node labels and relationship types | INTEGRATE | |
| UNWIND batch writes | INTEGRATE | |
| MERGE idempotent node and relationship upserts | INTEGRATE | |
| managed write transactions | INTEGRATE | |
| per-transaction timeout | INTEGRATE | |
| bounded transient-error retry | INTEGRATE | |
| context cancellation for connectivity, reads, writes, and close | INTEGRATE | |
| typed Neo4j error classification mapped to secret-safe result codes | INTEGRATE | |
| namespace lock and pending-generation manifest transactions | INTEGRATE | |
| atomic active-generation switch and exact-owner stale cleanup | INTEGRATE | |
| driver, session, and result-consumer cleanup | INTEGRATE | |
| routing-cluster discovery | INTEGRATE | Supported through the configured Neo4j URI and official driver rather than custom routing logic. |
| bookmarks and causal chaining across independent commands | OPT-OUT | Each push is a self-contained, manually invoked snapshot replacement whose own transactions and final activation define consistency. |
| impersonated users | OPT-OUT | Authentication is fixed by the explicitly named machine profile; per-request identity switching is outside the projection contract. |
| reactive API | OPT-OUT | The Go driver integration uses context-aware synchronous managed transactions and bounded batches. |
| CDC, subscriptions, and continuous synchronization | OPT-OUT | Explicitly deferred by the phase boundary; this phase is one-way manual materialization only. |
| reverse reads used as authoritative Gortex data | OPT-OUT | SQLite remains the sole authority and Neo4j target reads are limited to capability, constraint, manifest, lock, and census checks. |
