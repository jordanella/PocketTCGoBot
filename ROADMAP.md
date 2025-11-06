# PocketTCGoBot - Development Roadmap

This document outlines the development roadmap for transitioning from prototype to production-ready bot and achieving feature parity with the legacy AHK implementation.

## Current Status: Early Prototype

**Core Infrastructure:** ✅ Complete
**Basic Functionality:** 🚧 In Progress
**Feature Parity:** ⏳ Planned
**Polish & Testing:** ⏳ Planned

---

## Phase 1: Core Parity (Current Phase)

**Goal:** Match all functionality of the AHK bot with improved reliability

### 1.1 Action Library Completion 🚧

**Priority:** HIGH
**Status:** In Progress (30% complete)

- [x] Basic primitives (Click, Swipe, Sleep, Key)
- [x] CV actions (FindAndClickCenter, WaitFor)
- [x] Loop constructs (Until, While)
- [x] Wonder Pick basic flow
- [ ] Complete Wonder Pick error handling
- [ ] Pack opening sequences
- [ ] Mission navigation and completion
- [ ] Home navigation (GoHome)
- [ ] Shop navigation (GoToShop)
- [ ] Settings navigation
- [ ] Friend interactions
- [ ] Daily login flow
- [ ] Tutorial skip sequences

**Deliverable:** All AHK bot routines reimplemented in Go

### 1.2 Template Refactoring 🚧

**Priority:** HIGH
**Status:** Not Started (0% complete)

- [ ] Audit all template string references in code
- [ ] Replace hardcoded strings with `templates.*` references
- [ ] Ensure all templates defined in `pkg/templates/templates.go`
- [ ] Verify all template PNGs exist in `bin/templates/`
- [ ] Add missing templates from AHK version
- [ ] Document template naming conventions
- [ ] Create template testing utility

**Deliverable:** Zero hardcoded template strings, all use definitions

### 1.3 Error Handling & Recovery 🚧

**Priority:** HIGH
**Status:** Architecture in place (20% complete)

- [x] Error monitor structure
- [x] Health checker structure
- [ ] Implement error handlers
- [ ] Screen stuck detection
- [ ] App crash recovery
- [ ] Network error handling
- [ ] ADB disconnection recovery
- [ ] Automatic restart on critical errors
- [ ] Error logging and reporting
- [ ] Graceful degradation strategies

**Deliverable:** Bot can recover from common errors without manual intervention

### 1.4 Multi-Instance Coordination 🚧

**Priority:** MEDIUM
**Status:** Stubbed (10% complete)

- [x] Coordinator package structure
- [ ] Account pool management
- [ ] Instance lifecycle coordination
- [ ] Load balancing across instances
- [ ] Synchronized operations (e.g., Wonder Pick groups)
- [ ] Instance failure handling
- [ ] Resource sharing (accounts, statistics)

**Deliverable:** Manage 5+ instances simultaneously with shared account pool

### 1.5 Testing & Validation ⏳

**Priority:** MEDIUM
**Status:** Not Started (0% complete)

- [ ] Unit tests for action primitives
- [ ] Unit tests for CV service
- [ ] Integration tests for ADB operations
- [ ] Mock implementations for testing
- [ ] Template matching accuracy tests
- [ ] Configuration validation tests
- [ ] End-to-end bot flow tests
- [ ] Performance benchmarks

**Deliverable:** 60%+ test coverage with passing CI

---

## Phase 2: Enhanced Features

**Goal:** Improve upon AHK implementation with features not previously possible

### 2.1 Card Recognition & Logging 📋

**Priority:** HIGH
**Status:** Foundation in place (15% complete)

- [x] Card border detection structure
- [x] Basic rarity detection
- [ ] Accurate card rarity classification
- [ ] Special card detection (Crown, Immersive, Shiny)
- [ ] Card name recognition (requires OCR)
- [ ] Card image extraction and storage
- [ ] Duplicate detection
- [ ] Collection tracking
- [ ] Card statistics and analytics

**Deliverable:** Comprehensive card logging with accurate rarity detection

### 2.2 OCR Engine 📋

**Priority:** HIGH
**Status:** Placeholder (0% complete)

**Approach:** Pure Go implementation (no Tesseract dependency)

- [ ] Character template generation
- [ ] Character segmentation algorithm
- [ ] Template matching for characters
- [ ] Number recognition (shinedust count)
- [ ] Text recognition (card names)
- [ ] Error message detection
- [ ] Multi-language support (EN, CN, JP, etc.)
- [ ] OCR accuracy validation
- [ ] Performance optimization

**Deliverable:** Read shinedust counts and card names with 95%+ accuracy

### 2.3 Database Integration 📋

**Priority:** MEDIUM
**Status:** Placeholder (0% complete)

**Technology:** SQLite (embedded, no external dependencies)

- [ ] Schema design (accounts, cards, runs, statistics)
- [ ] Migration system
- [ ] Account metadata storage
- [ ] Card collection database
- [ ] Run history and statistics
- [ ] Performance metrics tracking
- [ ] Query API for GUI
- [ ] Backup and export functionality
- [ ] Data pruning and archival

**Deliverable:** Persistent storage for all bot data with query capabilities

### 2.4 Advanced Pack Logic 📋

**Priority:** MEDIUM
**Status:** Not Started (0% complete)

- [ ] Smart pack selection based on collection
- [ ] Pack priority weighting
- [ ] Event pack handling
- [ ] Limited-time pack detection
- [ ] Pack opening statistics
- [ ] Optimal pack timing (daily resets, etc.)
- [ ] Pack value estimation

**Deliverable:** Intelligent pack opening based on user preferences and statistics

### 2.5 Enhanced GUI 📋

**Priority:** LOW
**Status:** Basic implementation (40% complete)

- [x] Multi-tab interface
- [x] Real-time status updates
- [x] Configuration editing
- [ ] Live preview of emulator screens
- [ ] Statistics dashboard with charts
- [ ] Card collection browser
- [ ] Advanced filtering and sorting
- [ ] Export/import configurations
- [ ] Theme customization
- [ ] Notification center
- [ ] Real-time logs with filtering

**Deliverable:** Professional GUI with comprehensive monitoring and control

---

## Phase 3: Community Features

**Goal:** Enable collaboration and social features for Wonder Pick groups

### 3.1 Discord Integration 📋

**Priority:** MEDIUM
**Status:** Placeholder (0% complete)

- [ ] Webhook client implementation
- [ ] Card pull notifications
- [ ] Error/alert notifications
- [ ] Wonder Pick group coordination
- [ ] Showcase sharing
- [ ] Statistics sharing
- [ ] Bot status updates
- [ ] Account export on S4T triggers
- [ ] Rich embed formatting
- [ ] Rate limiting and error handling

**Deliverable:** Full Discord integration for notifications and coordination

### 3.2 Save for Trade (S4T) Automation 📋

**Priority:** MEDIUM
**Status:** Configuration in place (20% complete)

- [x] S4T configuration options
- [ ] Trigger detection (valuable cards)
- [ ] Account export on trigger
- [ ] Discord notification with card details
- [ ] Automatic account backup
- [ ] S4T statistics tracking
- [ ] Configurable S4T criteria
- [ ] Manual S4T trigger option

**Deliverable:** Automatic account preservation on valuable card pulls

### 3.3 Wonder Pick Group Features 📋

**Priority:** LOW
**Status:** Not Started (0% complete)

- [ ] Group coordination via Discord
- [ ] Showcase timing coordination
- [ ] Thanks tracking and reciprocation
- [ ] Group statistics
- [ ] Optimal Wonder Pick selection
- [ ] Group event coordination

**Deliverable:** Seamless Wonder Pick group collaboration

### 3.4 Web Dashboard (Future) 📋

**Priority:** LOW
**Status:** Not Started (0% complete)

- [ ] Web server with REST API
- [ ] Real-time status monitoring
- [ ] Remote bot control
- [ ] Statistics visualization
- [ ] Account management
- [ ] Multi-user support
- [ ] Mobile-responsive design

**Deliverable:** Web-based dashboard for remote monitoring and control

---

## Phase 4: Polish & Production

**Goal:** Production-ready software with professional quality

### 4.1 Documentation Completion 📋

**Priority:** MEDIUM
**Status:** Foundation complete (60% complete)

- [x] README.md
- [x] CONTRIBUTING.md
- [x] ARCHITECTURE.md
- [x] SETUP.md
- [x] API.md
- [ ] Video tutorials
- [ ] Troubleshooting guide expansion
- [ ] FAQ section
- [ ] Example configurations
- [ ] Migration guide from AHK
- [ ] Performance tuning guide

**Deliverable:** Comprehensive documentation for users and developers

### 4.2 Cross-Platform Support 📋

**Priority:** LOW
**Status:** Windows only (20% complete)

- [x] Windows support (primary)
- [ ] Linux testing and fixes
- [ ] macOS testing and fixes
- [ ] Multi-platform CV capture implementations
- [ ] Platform-specific installers
- [ ] Docker support for headless operation

**Deliverable:** Verified support for Windows, Linux, and macOS

### 4.3 Performance Optimization 📋

**Priority:** MEDIUM
**Status:** Not Started (0% complete)

- [ ] Template matching performance profiling
- [ ] Frame capture optimization
- [ ] Memory usage optimization
- [ ] CPU usage reduction
- [ ] Cache tuning
- [ ] Parallel processing where applicable
- [ ] Startup time optimization

**Deliverable:** 50%+ performance improvement over baseline

### 4.4 Security & Privacy 📋

**Priority:** HIGH
**Status:** Basic measures (30% complete)

- [x] Account XML gitignore protection
- [ ] Encryption for stored accounts
- [ ] Secure Discord webhook storage
- [ ] Privacy-focused logging (no sensitive data)
- [ ] Account data sanitization for sharing
- [ ] Security audit
- [ ] Responsible disclosure policy

**Deliverable:** Secure handling of all sensitive data

### 4.5 Packaging & Distribution 📋

**Priority:** MEDIUM
**Status:** Not Started (0% complete)

- [ ] Automated build pipeline (CI/CD)
- [ ] Windows installer (.msi or .exe)
- [ ] Linux packages (.deb, .rpm)
- [ ] macOS package (.dmg)
- [ ] Portable/standalone builds
- [ ] Auto-update functionality
- [ ] Release notes automation
- [ ] Version management

**Deliverable:** Professional installers and distribution system

---

## Deprecation Timeline: AHK Bot

### Milestone 1: Feature Parity (Phase 1 Complete)
- All AHK bot routines reimplemented
- Error handling matches or exceeds AHK
- Multi-instance support stable

**Target:** Q2 2025

### Milestone 2: Production Release (Phase 4 Complete)
- Comprehensive testing complete
- Documentation complete
- Installers available
- Community adoption begins

**Target:** Q3 2025

### Milestone 3: AHK Deprecation
- Official announcement of AHK bot deprecation
- Migration guide published
- Support for AHK version ends
- Archive AHK codebase

**Target:** Q4 2025

---

## Success Metrics

### Technical Metrics
- [ ] 60%+ test coverage
- [ ] 95%+ uptime for 24-hour runs
- [ ] <5% error rate on actions
- [ ] <100ms average template matching time
- [ ] Support for 10+ simultaneous instances

### Community Metrics
- [ ] 100+ active users
- [ ] 10+ contributors
- [ ] 50+ GitHub stars
- [ ] Active Discord community

### Feature Completion
- [ ] 100% AHK feature parity
- [ ] Card recognition implemented
- [ ] Database logging functional
- [ ] Discord integration complete

---

## Contributing to Roadmap

We welcome community input on priorities and feature requests!

**Process:**
1. Open a GitHub Discussion for major features
2. Create an issue for specific tasks
3. Comment on existing roadmap items
4. Submit PRs for roadmap tasks

**Current High-Priority Help Needed:**
- Template refactoring (replace hardcoded strings)
- Unit test writing
- Action library completion
- Cross-platform testing
- Documentation improvements

---

## Version Milestones

- **v0.1.0** - Initial prototype (current)
- **v0.2.0** - Action library complete
- **v0.3.0** - Template refactoring complete
- **v0.4.0** - Error handling complete
- **v0.5.0** - Multi-instance coordination
- **v1.0.0** - Feature parity with AHK (Phase 1 complete)
- **v1.5.0** - Card recognition and OCR (Phase 2)
- **v2.0.0** - Discord integration (Phase 3)
- **v2.5.0** - Production polish (Phase 4)
- **v3.0.0** - Enhanced functionality

---

**Last Updated:** 2025-01-05
**Current Version:** v0.1.0-prototype
**Next Milestone:** v0.2.0 - Action Library Complete
