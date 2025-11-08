# PocketTCGoBot - Development Roadmap v2.0

**Status:** ✅ **Prototype Complete** - Ready for Domain-Specific Development

This document outlines the next phase of development now that the core bot infrastructure is complete and functioning.

---

## Current Status (2025-11-08)

**Core Infrastructure:** ✅ **100% Complete**
**Bot Management:** ✅ **100% Complete**
**Health & Resilience:** ✅ **100% Complete**
**Domain Organization:** ✅ **100% Complete**
**Developer Experience:** ✅ **100% Complete**

**Current Version:** v0.5.0-prototype
**Next Milestone:** v1.0.0 - Production-Ready Bot with Pokemon TCG Pocket Routines

---

## ✅ COMPLETED PHASES (Phases 1-4)

### Phase 1: Instance Lifecycle Controls ✅
**Status:** Complete
**Completion Date:** 2025-11-08

**Implemented Features:**
- ✅ Individual bot control buttons (Pause/Resume/Stop/Restart)
- ✅ Per-bot status indicators with color-coded circles
- ✅ Real-time status polling (500ms interval)
- ✅ Restart functionality with last routine tracking
- ✅ Button state management based on bot state
- ✅ Status labels updating automatically

**Files Modified:**
- `internal/gui/bot_launcher.go` - Individual controls, status polling
- `internal/bot/bot.go` - Last routine tracking
- `internal/bot/manager.go` - Restart methods

---

### Phase 2: Bot Health & Resilience ✅
**Status:** Complete
**Completion Date:** 2025-11-08

**Implemented Features:**
- ✅ Health monitoring with ADB connection checks
- ✅ Device responsiveness detection
- ✅ Configurable health check intervals (10s default)
- ✅ Stuck detection with 30s timeout
- ✅ Auto-restart on failure with exponential backoff
- ✅ Restart policy configuration (maxRetries, backoff, delays)
- ✅ Sentry activity monitoring with comprehensive metrics
- ✅ Sentry health checks and consecutive error tracking

**Files Modified:**
- `internal/monitor/health_checker.go` - Full implementation
- `internal/bot/config.go` - RestartPolicy struct
- `internal/bot/manager.go` - ExecuteWithRestart method
- `internal/actions/sentry_metrics.go` - NEW FILE
- `internal/actions/sentry_engine.go` - Metrics integration

---

### Phase 3: Domain Script Organization ✅
**Status:** Complete
**Completion Date:** 2025-11-08

**Implemented Features:**
- ✅ Recursive subdirectory support for routines
- ✅ Namespace system (e.g., "combat/battle_loop")
- ✅ Domain folders created: combat/, farming/, navigation/, error_handling/, examples/
- ✅ README.md documentation for each domain
- ✅ Template starter files (_template.yaml) for each domain
- ✅ Existing routines migrated to appropriate domains
- ✅ Comprehensive naming conventions documented

**Files Modified:**
- `internal/actions/routine_registry.go` - Recursive scanning, namespace methods
- `internal/gui/bot_launcher.go` - Namespace grouping display

**Created Files:**
- `routines/README.md` - Main documentation
- `routines/combat/README.md` + `_template.yaml`
- `routines/farming/README.md` + `_template.yaml`
- `routines/navigation/README.md` + `_template.yaml`
- `routines/error_handling/README.md` + `_template.yaml`
- `routines/examples/README.md`

---

### Phase 4: Developer Experience ✅
**Status:** Complete
**Completion Date:** 2025-11-08

**Implemented Features:**
- ✅ Hot Reload for routines (reload button in GUI)
- ✅ Hot Reload for templates (reload button in GUI)
- ✅ Visual feedback on reload with success dialogs
- ✅ Real-time variable inspector (expandable accordion per bot)
- ✅ Variable display updates every 500ms
- ✅ Dynamic config editor GUI with form generation
- ✅ Type-based input widgets (text, number, checkbox, dropdown)
- ✅ Full validation (required fields, min/max, type checking)
- ✅ Config override system (per-bot customization)

**Files Modified:**
- `internal/gui/bot_launcher.go` - Reload buttons, variable inspector, config editor
- `internal/bot/bot.go` - GetAllVariables method
- `internal/bot/manager.go` - GetBotVariables, ReloadRoutines, ReloadTemplates

---

## 🎯 PHASE 5: Pokemon TCG Pocket Domain Routines (CURRENT FOCUS)

**Goal:** Create production-ready routines for Pokemon TCG Pocket automation

**Priority:** 🔴 **CRITICAL** (Revenue-generating capability)
**Effort:** High (2-3 weeks)
**Status:** Not Started

### 5.1 Template Library for Pokemon TCG Pocket 🖼️

**Priority:** 🔴 CRITICAL
**Effort:** Medium (1 week)

**Implementation Tasks:**
- [ ] Screenshot and create templates for main UI elements
  - [ ] Home screen buttons (Shop, Battles, Packs, etc.)
  - [ ] Battle start/end screens
  - [ ] Reward collection buttons
  - [ ] Navigation elements (back button, close buttons)
  - [ ] Menu buttons and tabs
- [ ] Create template YAML definitions in config/templates/
- [ ] Organize templates by screen/context
- [ ] Test template matching with various resolutions
- [ ] Document template naming conventions

**Deliverable:** 50+ templates covering core Pokemon TCG Pocket UI

---

### 5.2 Navigation Routines 🧭

**Priority:** 🔴 CRITICAL
**Effort:** Small (3-4 days)

**Implementation Tasks:**
- [ ] `navigation/go_to_home.yaml` - Return to home from any screen
- [ ] `navigation/home_to_shop.yaml` - Navigate home → shop
- [ ] `navigation/home_to_battle.yaml` - Navigate home → battle selection
- [ ] `navigation/home_to_packs.yaml` - Navigate home → pack opening
- [ ] `navigation/home_to_missions.yaml` - Navigate home → daily missions
- [ ] `navigation/dismiss_popups.yaml` - Close any popup/ad
- [ ] Test all navigation paths with multiple devices

**Deliverable:** 6+ navigation routines covering major screens

---

### 5.3 Pack Opening Routines 🎁

**Priority:** 🟡 HIGH
**Effort:** Medium (4-5 days)

**Implementation Tasks:**
- [ ] `farming/open_free_packs.yaml` - Open all available free packs
- [ ] `farming/mass_pack_opening.yaml` - Open N packs with counter
- [ ] Error handling for "no packs available"
- [ ] Card collection confirmation
- [ ] Integration with navigation routines
- [ ] Config parameters: max_packs, pack_type

**Deliverable:** Pack opening automation with configurable limits

---

### 5.4 Daily Mission Completion 📋

**Priority:** 🟡 HIGH
**Effort:** Medium (4-5 days)

**Implementation Tasks:**
- [ ] `farming/complete_daily_missions.yaml` - Automated mission completion
- [ ] Mission type detection (battle, pack opening, etc.)
- [ ] Conditional execution based on mission requirements
- [ ] Reward collection
- [ ] Mission status tracking with variables
- [ ] Config parameters: mission_types, priority_order

**Deliverable:** Automated daily mission system

---

### 5.5 Battle Routines ⚔️

**Priority:** 🟢 MEDIUM
**Effort:** High (1 week)

**Implementation Tasks:**
- [ ] `combat/ai_battle_basic.yaml` - Simple AI battle execution
- [ ] Card selection logic (basic strategy)
- [ ] Energy management
- [ ] Attack execution
- [ ] Win/loss detection
- [ ] Reward collection
- [ ] Battle loop with configurable max_battles
- [ ] Config parameters: max_duration, battle_mode

**Deliverable:** Basic battle automation

---

### 5.6 Resource Farming Loops 💰

**Priority:** 🟡 HIGH
**Effort:** Medium (4-5 days)

**Implementation Tasks:**
- [ ] `farming/farm_coins.yaml` - Coin farming loop
- [ ] `farming/farm_experience.yaml` - XP farming
- [ ] `farming/farm_battle_rewards.yaml` - Automated battle rewards
- [ ] Resource full detection
- [ ] Dynamic delay between runs
- [ ] Config parameters: max_runs, run_delay, target_amount

**Deliverable:** Configurable farming routines

---

### 5.7 Error Handling & Recovery 🛡️

**Priority:** 🔴 CRITICAL
**Effort:** Medium (4-5 days)

**Implementation Tasks:**
- [ ] `error_handling/connection_lost.yaml` - Reconnection logic
- [ ] `error_handling/game_crashed.yaml` - Game restart
- [ ] `error_handling/popup_ads.yaml` - Ad dismissal
- [ ] `error_handling/unexpected_screen.yaml` - Recovery navigation
- [ ] `error_handling/resource_full.yaml` - Storage full handling
- [ ] Sentry configuration for all main routines
- [ ] Test error scenarios thoroughly

**Deliverable:** Robust error recovery system

---

## 🎯 PHASE 6: Production Hardening

**Goal:** Make bot production-ready with safety, logging, and monitoring

**Priority:** 🟡 HIGH
**Effort:** Medium (1-2 weeks)
**Status:** Not Started

### 6.1 Comprehensive Logging 📝

**Priority:** 🟡 HIGH
**Effort:** Small (2-3 days)

**Implementation Tasks:**
- [ ] Structured logging with severity levels
- [ ] Log rotation and size limits
- [ ] Per-bot log files (bot_1.log, bot_2.log)
- [ ] Action execution logging
- [ ] Error stack traces
- [ ] Performance metrics logging
- [ ] Log viewer in GUI (optional)

**Files to Create:**
- `internal/logging/logger.go` - Logging system
- `logs/` directory structure

---

### 6.2 Safety Mechanisms 🔒

**Priority:** 🔴 CRITICAL
**Effort:** Small (2-3 days)

**Implementation Tasks:**
- [ ] Rate limiting (max actions per minute)
- [ ] Session duration limits (max 4 hours continuous)
- [ ] Random delays between actions (human-like behavior)
- [ ] Detection avoidance patterns
- [ ] Emergency stop mechanism
- [ ] Resource usage monitoring

**Benefit:** Prevent account bans and detection

---

### 6.3 Statistics & Reporting 📊

**Priority:** 🟢 MEDIUM
**Effort:** Medium (4-5 days)

**Implementation Tasks:**
- [ ] Session statistics tracking
  - [ ] Total packs opened
  - [ ] Battles won/lost
  - [ ] Resources collected
  - [ ] Missions completed
- [ ] CSV export for statistics
- [ ] Daily/weekly summaries
- [ ] GUI dashboard for stats
- [ ] Performance metrics (actions/minute, uptime)

**Deliverable:** Comprehensive tracking and reporting

---

### 6.4 Configuration Management 📐

**Priority:** 🟢 MEDIUM
**Effort:** Small (2-3 days)

**Implementation Tasks:**
- [ ] Global configuration file (bot_config.yaml)
- [ ] Per-bot configuration overrides
- [ ] Hot reload for configuration changes
- [ ] Configuration validation
- [ ] Default configuration templates
- [ ] GUI for editing global config

**Deliverable:** Centralized, flexible configuration system

---

## 🎯 PHASE 7: Advanced Features (Optional)

**Goal:** Enhanced automation and user experience

**Priority:** 🟢 LOW
**Effort:** Variable
**Status:** Planned

### 7.1 Advanced Battle AI 🧠

- [ ] Card synergy detection
- [ ] Deck-specific strategies
- [ ] Meta-game adaptation
- [ ] Win rate optimization
- [ ] Machine learning integration (future)

---

### 7.2 Card Collection Tracking 🎴

- [ ] Database for card inventory
- [ ] Card rarity tracking
- [ ] Collection completion percentage
- [ ] Duplicate card detection
- [ ] OCR for card recognition

---

### 7.3 Account Management 👥

- [ ] Multi-account rotation
- [ ] Account session limits
- [ ] Account-specific routines
- [ ] Login/logout automation
- [ ] Account health monitoring

---

### 7.4 Discord Integration 🔔

- [ ] Status notifications
- [ ] Error alerts
- [ ] Statistics reporting
- [ ] Remote control commands
- [ ] Multi-bot monitoring dashboard

---

### 7.5 Routine Marketplace 🏪

- [ ] Community-shared routines
- [ ] Routine rating system
- [ ] Version control for routines
- [ ] Automatic routine updates
- [ ] Routine testing framework

---

## 📋 IMPLEMENTATION PRIORITIES

### Immediate (Next 2 Weeks)
1. 🔴 **Phase 5.1** - Template Library (Critical foundation)
2. 🔴 **Phase 5.2** - Navigation Routines (Core functionality)
3. 🔴 **Phase 5.7** - Error Handling (Stability)

### Short-term (Weeks 3-4)
4. 🟡 **Phase 5.3** - Pack Opening
5. 🟡 **Phase 5.4** - Daily Missions
6. 🟡 **Phase 5.6** - Resource Farming

### Medium-term (Weeks 5-6)
7. 🟢 **Phase 5.5** - Battle Routines
8. 🟡 **Phase 6.1** - Logging System
9. 🔴 **Phase 6.2** - Safety Mechanisms

### Long-term (Weeks 7-8)
10. 🟢 **Phase 6.3** - Statistics & Reporting
11. 🟢 **Phase 6.4** - Configuration Management
12. 🟢 **Phase 7.x** - Advanced Features (as needed)

---

## 🎯 SUCCESS METRICS

### Phase 5 Success Criteria
- [ ] Can navigate to any major screen from home
- [ ] Can open 10 packs automatically
- [ ] Can complete 5 daily missions without manual intervention
- [ ] Can farm resources for 1 hour continuously
- [ ] Can execute 10 battles with basic AI
- [ ] Error recovery works for 95% of common errors
- [ ] 0 crashes in 4-hour test run

### Phase 6 Success Criteria
- [ ] All actions logged with timestamps
- [ ] 24-hour continuous operation without detection
- [ ] Statistics exported to CSV successfully
- [ ] Configuration changes applied without restart
- [ ] Emergency stop functional within 2 seconds

### Production Readiness Criteria (v1.0)
- [ ] Template library covers 95% of UI elements
- [ ] 20+ production-ready routines
- [ ] Error recovery rate > 95%
- [ ] 48-hour stability test passed
- [ ] Safety mechanisms prevent detection
- [ ] Full logging and monitoring
- [ ] Documentation complete

---

## 🏗️ ARCHITECTURE STATUS

### Completed ✅
- **41 Actions** - Comprehensive action system
- **Routine System** - YAML-based with eager loading
- **Sentry Supervision** - Parallel monitoring
- **Template Registry** - Image caching
- **Variable System** - Per-instance stores
- **Config System** - User-configurable parameters
- **Multi-Instance Manager** - Shared registries
- **Health Monitoring** - ADB and device checks
- **Auto-Restart** - Exponential backoff
- **Hot Reload** - Templates and routines
- **Variable Inspector** - Real-time debugging
- **Config Editor** - Dynamic form generation
- **Domain Organization** - Namespaced routines

### In Progress 🚧
- **Template Library** - Pokemon TCG Pocket specific
- **Domain Routines** - Game-specific automation

### Planned 📋
- **Logging System** - Comprehensive logging
- **Safety Mechanisms** - Rate limiting, detection avoidance
- **Statistics Tracking** - Session metrics

---

## 📊 VERSION MILESTONES

- ✅ **v0.1.0** - Core infrastructure complete
- ✅ **v0.2.0** - Individual bot controls (Phase 1)
- ✅ **v0.3.0** - Health & resilience (Phase 2)
- ✅ **v0.4.0** - Domain script library (Phase 3)
- ✅ **v0.5.0** - Developer experience polish (Phase 4)
- 🚧 **v0.6.0** - Pokemon TCG Pocket templates
- 🚧 **v0.7.0** - Core automation routines
- 🚧 **v0.8.0** - Error handling & recovery
- 📋 **v0.9.0** - Production hardening (logging, safety)
- 📋 **v1.0.0** - **Production Release** - Full Pokemon TCG Pocket automation

---

## 🚀 RECOMMENDED DEVELOPMENT SCHEDULE

### Weeks 1-2: Template & Navigation Foundation
**Goal:** Create template library and basic navigation

- **Days 1-5:** Screenshot and create Pokemon TCG Pocket templates
- **Days 6-8:** Build navigation routines (home, shop, battles, etc.)
- **Days 9-10:** Error handling foundation (connection loss, popups)

**Deliverable:** Can navigate to any screen and recover from common errors

---

### Weeks 3-4: Core Automation
**Goal:** Implement pack opening and daily missions

- **Days 11-15:** Pack opening automation with error handling
- **Days 16-20:** Daily mission completion system
- **Days 21-24:** Resource farming loops (coins, XP)
- **Day 25:** Integration testing

**Deliverable:** Fully automated daily routine execution

---

### Weeks 5-6: Battle System & Production Features
**Goal:** Battle automation and production hardening

- **Days 26-32:** Basic battle AI and execution
- **Days 33-37:** Logging system implementation
- **Days 38-40:** Safety mechanisms (rate limiting, delays)
- **Days 41-42:** Integration testing

**Deliverable:** Production-ready bot with battle capabilities

---

### Weeks 7-8: Polish & Release
**Goal:** Statistics, monitoring, and final testing

- **Days 43-47:** Statistics tracking and reporting
- **Days 48-50:** Configuration management
- **Days 51-54:** 48-hour stability testing
- **Days 55-56:** Documentation and release preparation

**Deliverable:** v1.0.0 Production Release

---

## 🎓 DEVELOPMENT BEST PRACTICES

### Routine Development
1. **Start with Templates**: Create all necessary templates first
2. **Test Navigation**: Ensure reliable navigation before automation
3. **Error Handling**: Add sentries from the start
4. **Incremental Testing**: Test each routine in isolation
5. **Config Parameters**: Make routines configurable for flexibility
6. **Documentation**: Document routine behavior and config options

### Template Creation
1. **High Quality Screenshots**: Use native resolution
2. **Multiple Variations**: Capture different states (pressed, unpressed, etc.)
3. **Threshold Testing**: Test matching thresholds thoroughly
4. **Region Definition**: Use regions to speed up matching
5. **Naming Convention**: Descriptive names (button_battle_start, icon_shop, etc.)

### Safety Considerations
1. **Random Delays**: Add variability to all timing
2. **Rate Limiting**: Don't exceed human-possible speeds
3. **Session Limits**: Maximum 4-hour continuous operation
4. **Pattern Variation**: Don't repeat exact same sequence
5. **Error Recovery**: Always have fallback plans

---

## 🎯 DEFERRED FEATURES

These features are interesting but not critical for v1.0:

- ❌ Routine versioning and rollback
- ❌ Template visual editor
- ❌ Step-through debugger for routines
- ❌ Routine marketplace
- ❌ Load balancing across devices
- ❌ Advanced ML-based card recognition
- ❌ Cross-platform support (focus on Windows first)
- ❌ Mobile app integration
- ❌ Web dashboard

These can be revisited post-v1.0 based on user demand.

---

## 📈 RISK ASSESSMENT

### High Risk Items
1. **Detection Risk** - Implement safety mechanisms early
2. **Template Matching Failures** - Comprehensive template library needed
3. **Game Updates Breaking Templates** - Version templates, quick update process
4. **ADB Stability** - Health monitoring and auto-restart critical

### Mitigation Strategies
- Start with conservative automation (lower speed, longer delays)
- Create template variants for different resolutions
- Implement template versioning system
- Comprehensive error logging for debugging
- Community testing with multiple devices

---

**Last Updated:** 2025-11-08
**Current Version:** v0.5.0-prototype
**Next Milestone:** v0.6.0 - Pokemon TCG Pocket Template Library
**Focus:** Phase 5 - Pokemon TCG Pocket Domain Routines
**Target Release:** v1.0.0 by End of Week 8

---

## 🙏 ACKNOWLEDGMENTS

**Completed Work:**
- ✅ All core infrastructure (Phases 1-4)
- ✅ 41 actions implemented
- ✅ Sentry supervision system
- ✅ Health monitoring & auto-restart
- ✅ Variable inspection & config editor
- ✅ Hot reload capabilities
- ✅ Domain organization & templates

**Ready for Production Development:** The bot framework is complete and production-ready. All that remains is creating game-specific content (templates and routines) for Pokemon TCG Pocket.
