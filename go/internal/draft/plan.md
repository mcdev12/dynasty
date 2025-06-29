# Draft Orchestrator Implementation Plan

## Overview
This document outlines the implementation plan for a draft orchestrator that manages draft lifecycle, enforces state transitions, handles pick timeouts, and coordinates events using a database-backed scheduler approach.

## Goals
- Build a durable, scalable orchestrator for managing fantasy draft workflows
- Implement "pick-on-the-clock" timeouts without spawning thousands of goroutines
- Maintain consistency between draft state, picks, and published events
- Support crash recovery and horizontal scaling

## Architecture Components

### 1. Database Schema Updates
- Add `next_deadline TIMESTAMPTZ` column to `drafts` table
- Create partial index for efficient deadline queries
- Add columns for tracking retry attempts and last error (if needed)

### 2. Core Orchestrator Service
- Single service responsible for draft state management
- Implements state machine logic for draft transitions
- Manages timer scheduling and timeout handling
- Coordinates with existing publisher/listener infrastructure

### 3. Scheduler Component
- Single goroutine per instance that polls for expired deadlines
- Batch processes timeouts for efficiency
- Uses database-level locking to prevent duplicate processing

## Implementation Phases

### Phase 1: Database Schema & Repository Layer (Week 1)

**Tasks:**
1. Design and apply database migrations
   - Add `next_deadline` column to drafts table
   - Create partial index on (state, next_deadline) WHERE state = 'IN_PROGRESS'
   - Add audit columns (last_timeout_at, timeout_count)

2. Extend repository layer
   - Add methods for deadline management
   - Implement queries for scheduler (get next deadline, claim expired drafts)
   - Add transactional methods for state + deadline updates

3. Write repository tests
   - Test concurrent access patterns
   - Verify FOR UPDATE SKIP LOCKED behavior
   - Test deadline calculation logic

**Deliverables:**
- Migration files
- Updated repository interfaces and implementations
- Comprehensive test suite

### Phase 2: Orchestrator Core Logic (Week 2)

**Tasks:**
1. Implement state machine
   - Define valid state transitions
   - Create state transition handlers
   - Add validation for business rules

2. Build timeout handler
   - Auto-pick logic when user times out
   - Update next deadline after auto-pick
   - Publish appropriate events

3. Create orchestrator service interface
   - Methods for starting/pausing/resuming drafts
   - Pick processing (user picks and auto-picks)
   - Draft completion logic

4. Integration with existing services
   - Wire up with outbox pattern
   - Connect to existing publisher
   - Handle incoming events from listener

**Deliverables:**
- Orchestrator service implementation
- State machine with tests
- Integration with existing infrastructure

### Phase 3: Scheduler Implementation (Week 3)

**Tasks:**
1. Build scheduler loop
   - Query for next deadline
   - Implement smart sleep with context cancellation
   - Batch processing of expired drafts

2. Add resilience features
   - Exponential backoff for errors
   - Jitter to prevent thundering herd
   - Graceful shutdown handling

3. Implement worker pool
   - Bounded concurrency for timeout processing
   - Error handling and retry logic
   - Metrics for queue depth and processing time

4. Add observability
   - Prometheus metrics for scheduler performance
   - Structured logging for debugging
   - Distributed tracing integration

**Deliverables:**
- Scheduler component
- Worker pool implementation
- Monitoring and metrics

### Phase 4: Testing & Hardening (Week 4)

**Tasks:**
1. End-to-end testing
   - Full draft lifecycle tests
   - Timeout scenario testing
   - Concurrent draft management

2. Chaos testing
   - Test crash recovery
   - Simulate network partitions
   - Database connection failures

3. Performance testing
   - Load test with thousands of concurrent drafts
   - Measure scheduler latency
   - Optimize query performance

4. Documentation
   - API documentation
   - Runbook for operations
   - Architecture decision records

**Deliverables:**
- Comprehensive test suite
- Performance benchmarks
- Documentation package

## Technical Decisions

### Why Database-Backed Scheduling?
- **Durability**: Survives crashes without losing timers
- **Simplicity**: No external dependencies (Redis, etc.)
- **Consistency**: Deadlines updated atomically with state
- **Scalability**: Can run multiple orchestrator instances

### Why Single Scheduler Loop?
- **Resource Efficiency**: One goroutine vs thousands
- **Predictable**: Easy to reason about and debug
- **Testable**: Can mock time and database queries

### Why FOR UPDATE SKIP LOCKED?
- **Concurrency Safe**: Multiple workers can process without conflicts
- **Performance**: Non-blocking row-level locking
- **PostgreSQL Native**: No additional infrastructure

## Risk Mitigation

### Risk: Clock Drift
**Mitigation**: Use database NOW() for all time calculations

### Risk: Scheduler Bottleneck
**Mitigation**: 
- Batch processing of timeouts
- Multiple orchestrator instances
- Efficient indexing

### Risk: Cascading Failures
**Mitigation**:
- Circuit breakers for external calls
- Bounded retry attempts
- Degraded mode operation

## Success Metrics
- **Scheduler Latency**: < 100ms between deadline and processing
- **Throughput**: Handle 10,000+ concurrent drafts
- **Reliability**: 99.9% timeout accuracy
- **Recovery Time**: < 30s after crash

## Dependencies
- PostgreSQL 12+ (for SKIP LOCKED support)
- Existing outbox pattern implementation
- Current publisher/listener infrastructure

## Future Enhancements
1. **Pluggable Strategies**: Different timeout rules per draft type
2. **Preview Mode**: Test draft configurations without affecting users
3. **Analytics Pipeline**: Stream draft events for analysis
4. **Multi-Region**: Support for geo-distributed drafts

## Timeline Summary
- **Week 1**: Database and repository layer
- **Week 2**: Core orchestrator logic
- **Week 3**: Scheduler implementation
- **Week 4**: Testing and hardening
- **Total**: 4 weeks to production-ready

## Open Questions
1. Should we support configurable timeout durations per draft?
2. How should we handle auto-pick strategies (BPA, position need, etc.)?
3. What metrics are most important for the product team?
4. Should we implement draft rollback capabilities?