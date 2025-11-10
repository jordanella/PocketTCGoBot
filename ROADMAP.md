# PocketTCGoBot - Development Roadmap

This document outlines the development roadmap for achieving a production-ready multi-instance bot with full lifecycle controls.

## Current Status: v0.1.0 - Core Infrastructure Complete

**Core Infrastructure:** ✅ Complete (Actions, Routines, Sentries, Registries, Orchestrator, Account Pools, Database)
**Orchestration System:** ✅ Complete (UUID isolation, database checkout mutex)
**Instance Management:** 🚧 In Progress (Orchestrator complete, GUI controls needed)
**Domain Scripts:** 🚧 In Progress (Infrastructure ready, Pokemon TCG Pocket routines needed)
**Production Ready:** ⏳ Planned (v1.0.0)

### Recent Completions (2025-11-10)

- ✅ Orchestration ID system with UUID per bot group
- ✅ Database checkout mutex for account conflict prevention
- ✅ Account pool system with SQL queries and filtering
- ✅ MVC architecture for templates, routines, and pools
- ✅ Complete ARCHITECTURE.md documentation
- ✅ 11 database migrations with proper versioning

---

## PHASE 1: Instance Lifecycle Controls (CRITICAL - Current Focus)
**Goal:** Launch multiple configurable instances with controls to pause, stop, restart

### 1.1 Individual Bot Control in GUI ⚡
**Priority:** 🔴 CRITICAL
**Status:** In Progress
**Effort:** Medium (1-2 days)

**Current Gap:** Bot launcher only has "Launch All" and "Stop All" buttons. Cannot control individual bots.

**Implementation Tasks:**
- [x] Analyze existing bot_launcher.go structure
- [ ] Add per-bot status display (Idle/Running/Paused/Stopped/Completed)
- [ ] Add individual Pause button for each bot
- [ ] Add individual Resume button for each bot
- [ ] Add individual Stop button for each bot
- [ ] Wire buttons to existing RoutineController methods
- [ ] Add visual state indicators (colors/icons)
- [ ] Update status labels in real-time

**Files to Modify:**
- `internal/gui/bot_launcher.go` - Add individual control handlers
- `internal/gui/controller.go` - Add state query endpoints

**Benefit:** Full independent control over each bot instance.

---

### 1.2 Restart Functionality ⚡
**Priority:** 🔴 CRITICAL
**Status:** Not Started
**Effort:** Small (4-8 hours)

**Current Gap:** No way to restart a bot that has stopped or completed.

**Implementation Tasks:**
- [ ] Add `lastRoutineName` field to Bot struct
- [ ] Add `GetLastRoutine()` method to Bot
- [ ] Implement `RestartBot(instance)` in Manager
- [ ] Add Restart button to GUI per-bot controls
- [ ] Handle restart of running bots (stop then restart)

**Files to Modify:**
- `internal/bot/bot.go` - Track last routine executed
- `internal/bot/manager.go` - Add RestartBot() method
- `internal/gui/bot_launcher.go` - Add restart handler and button

**Benefit:** Quick recovery without reconfiguration.

---

### 1.3 Real-time Status Updates ⚡
**Priority:** 🟡 HIGH
**Status:** Not Started
**Effort:** Small (4-8 hours)

**Current Gap:** GUI doesn't update automatically when bot state changes.

**Implementation Tasks:**
- [ ] Add status polling goroutine per bot config
- [ ] Poll bot state every 500ms
- [ ] Update status labels automatically
- [ ] Update button enabled/disabled states based on state
- [ ] Add status change callbacks to RoutineController (optional enhancement)

**Files to Modify:**
- `internal/gui/bot_launcher.go` - Add polling mechanism
- `internal/bot/routine_state.go` - Optional: Add state change callbacks

**Benefit:** Real-time visibility without manual refresh.

**Milestone:** End of Week 1 - Can launch, pause, resume, stop, restart individual bots with real-time status

---

## PHASE 2: Bot Health & Resilience (HIGH Priority)
**Goal:** Make bots robust and self-recovering

### 2.1 Health Monitoring Implementation 🏥
**Priority:** 🟡 HIGH
**Status:** Stubbed (health_checker.go exists)
**Effort:** Medium (1-2 days)

**Implementation Tasks:**
- [ ] Implement CheckADBConnection() - verify ADB responsive
- [ ] Implement CheckDeviceResponsive() - verify device alive
- [ ] Add StartMonitoring() with configurable interval
- [ ] Add onUnhealthy callback system
- [ ] Integrate with Bot.Initialize()
- [ ] Log health check failures

**Files to Modify:**
- `internal/bot/health_checker.go` - Implement all checks
- `internal/bot/bot.go` - Start health monitoring

**Benefit:** Automatic detection of disconnects and frozen devices.

---

### 2.2 Auto-Restart on Failure 🔄
**Priority:** 🟡 HIGH
**Status:** Not Started
**Effort:** Small (4 hours)

**Implementation Tasks:**
- [ ] Add RestartPolicy struct (enabled, maxRetries, backoffDelay)
- [ ] Implement ExecuteWithRestart() in Manager
- [ ] Add restart policy to bot configuration
- [ ] Add retry counter and exponential backoff
- [ ] Log restart attempts

**Files to Modify:**
- `internal/bot/manager.go` - Add restart policy logic
- `internal/bot/config.go` - Add restart policy config

**Benefit:** Resilience to transient failures.

---

### 2.3 Sentry Activity Monitoring 👁️
**Priority:** 🟢 MEDIUM
**Status:** Not Started
**Effort:** Small (4-8 hours)

**Implementation Tasks:**
- [ ] Add SentryMetrics struct to sentry_engine.go
- [ ] Track execution count, success/failure, timing
- [ ] Add GetMetrics() method to SentryEngine
- [ ] Create GUI endpoint for sentry metrics
- [ ] Display sentry status in bot launcher (expandable section)

**Files to Modify:**
- `internal/actions/sentry_engine.go` - Add metrics collection
- `internal/gui/bot_launcher.go` - Display sentry metrics

**Benefit:** Debug sentry behavior and verify sentries are working.

**Milestone:** End of Week 2 - Bots recover from failures, health monitoring active

---

## PHASE 3: Domain Script Organization (MEDIUM Priority)
**Goal:** Make development of domain-specific scripts easier

### 3.1 Routine Subdirectory Support 📁
**Priority:** 🟢 MEDIUM
**Status:** Not Started
**Effort:** Small (4 hours)

**Implementation Tasks:**
- [ ] Add recursive directory scanning to routine_registry.go
- [ ] Support namespacing (e.g., "combat/battle_loop")
- [ ] Update GUI to show routines grouped by folder
- [ ] Update documentation

**Files to Modify:**
- `internal/actions/routine_registry.go` - Recursive scanning

**Benefit:** Organize as `routines/combat/`, `routines/farming/`, etc.

---

### 3.2 Routine Library Scaffolding 📚
**Priority:** 🟢 MEDIUM
**Status:** Not Started
**Effort:** Small (2-4 hours)

**Implementation Tasks:**
- [ ] Create domain folders (combat/, farming/, navigation/, error_handling/)
- [ ] Add README.md to each domain with conventions
- [ ] Create _template.yaml starter files
- [ ] Move existing routines to appropriate domains
- [ ] Document naming conventions

**Benefit:** Faster development with clear patterns and examples.

**Milestone:** End of Week 3 - Organized routine library ready for Pokemon TCG Pocket scripts

---

## PHASE 4: Developer Experience (LOW Priority - Polish)
**Goal:** Improve development workflow

### 4.1 Hot Reload in GUI 🔄
**Priority:** 🟢 LOW
**Effort:** Very Small (2 hours)

- [ ] Add "Reload Routines" button to GUI (handler already exists!)
- [ ] Add "Reload Templates" button to GUI (handler already exists!)
- [ ] Add visual feedback on reload

---

### 4.2 Variable Inspector 🔍
**Priority:** 🟢 LOW
**Effort:** Small (4 hours)

- [ ] Add endpoint to get bot's variable store
- [ ] Display variables in expandable section per bot
- [ ] Update in real-time during execution

---

### 4.3 Config Editor GUI 📝
**Priority:** 🟢 LOW
**Effort:** Medium (1-2 days)

- [ ] Generate forms from routine config definitions
- [ ] Allow editing config values before launch
- [ ] Validate inputs against constraints

---

## COMPLETED FEATURES ✅

### Core Architecture (v0.1.0)
- ✅ **41 Actions** - Click, Swipe, CV, Loops, Variables, Conditionals, Account Management
- ✅ **Routine System** - YAML-based with eager loading registry and composition
- ✅ **Sentry Supervision** - Parallel monitoring routines with recovery actions
- ✅ **Template Registry** - Image caching with YAML definitions (236+ templates)
- ✅ **Variable System** - Per-instance stores with `${variable}` interpolation
- ✅ **Config System** - User-configurable parameters with runtime overrides
- ✅ **Bot Group Orchestrator** - Multi-instance coordination with shared registries
- ✅ **Routine Controller** - State machine (Idle/Running/Paused/Stopped/Completed)
- ✅ **Comprehensive Documentation** - 15+ markdown docs including complete ARCHITECTURE.md

### Orchestration System (v0.1.0)
- ✅ **Orchestration ID** - UUID per bot group for execution context isolation
- ✅ **Database Checkout Mutex** - Global account injection conflict prevention
- ✅ **Account Pool Manager** - SQL queries, manual include/exclude, watched paths
- ✅ **Pool Definitions** - Shared YAML templates with execution-specific pools
- ✅ **Progress Monitoring** - InitialAccountCount tracking for UI display
- ✅ **Stale Detection** - 10-minute timeout for crash recovery
- ✅ **Cleanup on Shutdown** - Release all account checkouts for orchestration

### Database Integration (v0.1.0)
- ✅ **SQLite Backend** - Account storage, routine executions, checkout tracking
- ✅ **Migration System** - 11 migrations with proper versioning
- ✅ **Migration 010** - orchestration_id in routine_executions table
- ✅ **Migration 011** - Checkout columns in accounts table
- ✅ **Account Checkout API** - Complete CRUD operations in account_checkout.go
- ✅ **Routine Execution Tracking** - Per-account lifecycle with orchestration context

### Registration Systems (Complete)
- ✅ **Action Registry** - 41 actions mapped and documented
- ✅ **Template Registry** - Dynamic YAML loading with caching
- ✅ **Routine Registry** - Metadata, validation, tag filtering, reload support
- ✅ **Pool Manager** - Pool definition registry with MVC architecture

### GUI Features (v0.1.0)
- ✅ **Multi-Tab Interface** - Dashboard, bot launcher, ADB test, config
- ✅ **Account Pool Wizard** - Visual query builder for pool definitions
- ✅ **Template Manager** - Load and cache templates from YAML
- ✅ **Routine Browser** - View, validate, and reload routines
- ✅ **Emulator Manager** - MuMu instance detection and management
- ✅ **Bot Group Launcher** - Launch/stop orchestration groups

---

## ARCHITECTURE STRENGTHS

✅ **Build-Execute Pattern** - Routines compiled once, executed many times
✅ **Shared Registries** - Memory efficient for multi-instance coordination
✅ **Thread-Safe State Management** - Atomic operations + mutexes
✅ **Two-Tier Conflict Resolution** - Orchestration ID + Database checkout mutex
✅ **Execution Context Isolation** - UUID per bot group prevents cross-contamination
✅ **Database Source of Truth** - Account checkout tracked in SQLite
✅ **Extensible Action System** - Easy to add new actions via registry
✅ **Clean Separation** - Instance state vs shared resources
✅ **Comprehensive Validation** - Early error detection with helpful messages
✅ **MVC Architecture** - Proper separation for templates, routines, pools

---

## NON-PRIORITIES (Not Needed for Prototype)

These are interesting but NOT needed for a functioning prototype:

- ❌ Bot Coordinator account injection (stubbed, not critical)
- ❌ Routine versioning
- ❌ Template visual editor
- ❌ Routine debugger (step-through)
- ❌ Routine marketplace
- ❌ Load balancing
- ❌ Circular sentry dependency detection
- ❌ Discord integration
- ❌ OCR engine
- ❌ Database logging
- ❌ Card recognition

---

## RECOMMENDED IMPLEMENTATION SCHEDULE

### Week 1: Minimum Viable Prototype ⚡
**Goal:** Independent bot control with full lifecycle management

- **Day 1-2:** Individual bot controls in GUI (#1.1) - CRITICAL
- **Day 3:** Restart functionality (#1.2) - CRITICAL
- **Day 4:** Status polling/real-time updates (#1.3) - HIGH
- **Day 5:** Testing multi-instance scenarios, bug fixes

**Deliverable:** Can launch 5 bots, independently pause/resume/stop/restart each one

---

### Week 2: Robustness 🏥
**Goal:** Bots recover from failures automatically

- **Day 6-7:** Health monitoring implementation (#2.1)
- **Day 8:** Auto-restart on failure (#2.2)
- **Day 9:** Sentry activity monitoring (#2.3)
- **Day 10:** Testing and refinement

**Deliverable:** Bots recover from ADB disconnects, restart on errors, sentry visibility

---

### Week 3: Domain Scripts 📚
**Goal:** Organized script library for Pokemon TCG Pocket

- **Day 11:** Routine subdirectory support (#3.1)
- **Day 12-13:** Create domain script libraries (#3.2)
- **Day 14-15:** Write domain-specific routines (Pokemon TCG Pocket)

**Deliverable:** Organized routine library with combat, farming, navigation scripts

---

### Week 4+: Polish ✨
**Goal:** Developer experience improvements

- Hot reload buttons
- Variable inspector
- Config editor GUI
- Additional domain scripts

---

## SUCCESS METRICS

### Week 1 Success Criteria
- [ ] Launch 6 bot instances simultaneously
- [ ] Pause bot #2 while others continue running
- [ ] Resume bot #2 without affecting others
- [ ] Stop bot #4 individually
- [ ] Restart bot #1 with same routine
- [ ] Real-time status updates without manual refresh

### Week 2 Success Criteria
- [ ] Bots automatically detect ADB disconnects
- [ ] Bots restart after transient errors (3 retry max)
- [ ] Sentry execution metrics visible in GUI
- [ ] 24-hour stability test with 5 bots

### Week 3 Success Criteria
- [ ] Routines organized in domain folders
- [ ] Pokemon TCG Pocket specific routines created
- [ ] Clear conventions documented
- [ ] Template routines for common patterns

---

## KNOWN GAPS (Documented for Future)

### Critical for Prototype
1. ❌ Individual bot GUI controls
2. ❌ Restart mechanism
3. ❌ Real-time status polling

### High Priority (Post-Prototype)
4. ❌ Health monitoring implementation
5. ❌ Auto-restart policy
6. ❌ Sentry metrics

### Medium Priority
7. ❌ Subdirectory support for routines
8. ❌ Domain script organization

### Low Priority
9. ❌ Hot reload GUI buttons
10. ❌ Variable inspector
11. ❌ Config editor GUI

---

## VERSION MILESTONES

- **v0.1.0** - Core infrastructure complete ✅ (CURRENT)
- **v0.2.0** - Individual bot controls (Week 1)
- **v0.3.0** - Health & resilience (Week 2)
- **v0.4.0** - Domain script library (Week 3)
- **v0.5.0** - Developer experience polish (Week 4+)
- **v1.0.0** - Functioning prototype ready for domain development

---

## LEGACY ROADMAP ITEMS (Deferred)

The following items from the original roadmap are deferred until after the functioning prototype is complete:

- Feature parity with AHK bot (Phase 1)
- Card recognition & logging (Phase 2)
- OCR engine (Phase 2)
- Database integration (Phase 2)
- Discord integration (Phase 3)
- Testing & validation (comprehensive)
- Cross-platform support
- Performance optimization
- Security & privacy hardening
- Packaging & distribution

These will be revisited once the core prototype is functioning and domain-specific scripts are being developed.

---

**Last Updated:** 2025-11-10
**Current Version:** v0.1.0
**Next Milestone:** v0.2.0 - Individual Bot Controls
**Current Focus:** Phase 1 - Instance Lifecycle Controls

### Version History

- **v0.1.0** (2025-11-10) - Core infrastructure complete
  - 41 actions, routine system, sentry engine
  - Orchestration ID system with UUID per bot group
  - Database checkout mutex for account conflicts
  - Account pool manager with SQL filtering
  - Complete ARCHITECTURE.md documentation
  - 11 database migrations
