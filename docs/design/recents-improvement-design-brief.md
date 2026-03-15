# Recents Tab Improvement - Design Brief

## Problem Statement

The Recents tab currently fails to support the user's mental model for triaging notifications, leading to three key pain points:

### 1. "Recent" Means Time-Based, Not Count-Based

**The Issue:**
- Current implementation uses a count-based limit: "latest 20 notifications, max 3 per source"
- Users hear a sound notification, jump to Recents, and expect to see what just happened
- Instead, they see notifications from a day ago because the "20 most recent" includes stale items
- This breaks the reactive workflow of quickly responding to new events

**User Quote:**
> "Look how it shows me the last 30 minutes, 1 hour, 30 minutes, last 30 minutes. Today I have a lot of messages but I have a limited message. They are not recent, right? Some of them are like a day old or like an hour old and I am not sure what you think about that."

**Example:**
User hears a sound notification now, opens Recents and sees:
- "Build failed" - 2 days ago
- "Tests passed" - 18 hours ago
- "Deployment started" - 1 day ago

These are NOT "recent" from the user's perspective, but they appear because they're in the "latest 20".

### 2. Noisy Sessions Dominate the List

**The Issue:**
- One session can generate 50+ notifications in an hour (e.g., a CI/CD pipeline)
- Current limit shows 3 notifications per source, so this noisy session dominates
- User sees: 3 from Session A, 3 from Session B, 0 from Sessions C, D, E, F, G...
- This doesn't give a good overview of which projects are active

**User Quote:**
> "Also I don't want to see too many of the same kind of project, right? Usually the messages are duplicated, right? It's the same message for a more interested in different messages from different projects, right?"

**Example:**
5 sessions active, but Session A is noisy:
- Session A: 50 notifications in last hour
- Session B: 10 notifications
- Session C: 5 notifications
- Session D: 2 notifications
- Session E: 1 notification

Current behavior: Shows 3 from A, 3 from B, 3 from C, 2 from D, 1 from E = 12 items
- Sessions A and B dominate the view
- Hard to see that Sessions D and E are even active

### 3. Poor Project Overview Across Sessions

**The Issue:**
- User needs to quickly scan which projects (tmux sessions) are active
- Current implementation doesn't optimize for diversity across sessions
- User can't tell at a glance which projects need attention
- This defeats the purpose of using Recents as a "project activity dashboard"

**User Context:**
- Each tmux session represents a project they're working on
- They need to see "which projects have activity right now?"
- Not just "what are the latest notifications?"
- The distinction matters: one is about projects (sessions), the other about events (notifications)

## User's Workflow

### Day-to-Day Context

The user works on multiple projects simultaneously, where each tmux session represents a project:

- **Session = Project:** Each tmux session is a project they're working on
- **Background Agents:** Agents run in the background doing their own things
- **Sound Notifications:** When a new message arrives, they hear a sound on their headphones
- **Reactive Triage:** Sound → Jump to app → Quick decision → Act or defer

### Core Workflow Pattern

```
1. Hear sound notification on headphones
   ↓
2. Jump to Recents tab (expecting recent activity)
   ↓
3. Quick scan of notifications
   - If urgent: "I need to handle this NOW"
     → Filter by session name
     → See what happened recently
     → Jump to session to fix
   - If not urgent: "This can wait"
     → Switch to All tab
     → Add to mental list of "projects to check later"
```

### Recents Tab: "What Just Happened?" (Reactive Mode)

**Purpose:**
- Quick triage of sound notifications
- Identify which projects are active right now
- Decide: handle now or defer to later

**Mental Model:**
- "What just triggered that sound?"
- "Which projects need attention right now?"
- "Can I handle this quickly, or should I defer?"

**Time Horizon:**
- Very short (minutes, maybe an hour)
- If I've been away for 3 hours, I expect to see 3 hours of activity
- If I've been away for 24 hours, I expect Recents to be empty (or very recent only)

**Usage Pattern:**
- Quick scan → Filter by session name → Jump to handle
- Filtering is key: "I can quickly filter to jump to that session"

### All Tab: "What's Going On?" (Proactive Mode)

**Purpose:**
- Project overview across all work
- Status check on ongoing projects
- Mental list of what to work on next

**Mental Model:**
- "What projects have I been working on?"
- "What's the overall status of my work?"
- "What should I prioritize next?"

**Time Horizon:**
- Longer (past days, past week)
- Messages from yesterday, 2 days ago, etc. are fine here
- This is where I go when I have time to do broader planning

**Usage Pattern:**
- Browse through → See which projects are active → Make decision about priorities

**User Quote:**
> "Sometimes I use this as my list for what projects are going on currently, right? I go to all and now the message in all the projects has a message, right? It's the project I have been working on in the past week, past days, right?"

### Key Insight: Duplicated Messages

The user mentioned that messages are often duplicated:

- Multiple similar notifications from the same session
- They want to see "different messages from different projects"
- They're interested in variety across sessions, not volume from one session

**User Quote:**
> "I'm limited by the number of items in the beginning. I think since it's recent it should be filtered by time."

This reinforces that "recent" is about **time window**, not **item count**.

### Key Constraint: Keep It Simple (Flat List)

When I suggested grouping by session with sections, the user clarified:

> "But remember I wanted to show this in a list, right? I don't want to have grouping or anything. It's just a list and then I can filter because one of the key aspects of this is that I can quickly filter to jump to that session."

**Critical Requirement:**
- Keep flat list (no grouping/collapsible sections)
- Filtering is the primary interaction for session investigation
- Quick filtering → Quick jumping is the core value prop

### Additional User Insight: Severity Awareness

During discussion, the user expressed interest in severity-aware behavior:

> "I like the idea of having the latest server. Maybe we can have sections for each, right? Each project has a box for info and a box for other categories. I'm not sure or maybe we can present the latest per category."

> "I like your idea of having the severity awareness but also don't do too much noise. Don't occupy all this space from the resets"

**Interpretation:**
- User wants severity awareness (don't miss important errors)
- But doesn't want noise (don't show everything)
- Interested in "latest per category" (latest error, latest warning, latest info)
- However, given the "flat list" constraint, this means:
  - Unfiltered: 1 item per session (severity-aware selection)
  - Filtered: Up to 10 items (can show mix of severity levels)

This aligns with the "1 per session, severity-aware" decision, while respecting the flat list requirement.

## Requirements

### Functional Requirements

#### Story 1: Time-Based Filtering (tmux-intray-nkv)

Recents tab only shows notifications within time window (default: 1h)

- Time window is configurable via `recents_time_window` config
- Valid time window values: "5m", "15m", "30m", "1h", "2h", "6h", "12h", "24h"
- When Recents is empty, show helpful message
- Message format: "No recent notifications (window: last 1 hour). Check All tab for older notifications."

#### Story 2: Per-Session Smart Selection - Unfiltered (tmux-intray-lkb)

Unfiltered Recents shows max 1 notification per session

- Selection priority: error > warning > info (show most recent of that level)
- Total unfiltered limit: 20 items
- Order by most recent activity first
- Tie-breaker: severity (error > warning > info)

#### Story 3: Filtered List Behavior (tmux-intray-2c4)

When filtering by session name, show up to 10 notifications

- Filtered notifications respect time window
- Filtered notifications ordered by timestamp (newest first)
- Show all if < 10 in time window
- Show empty message if no notifications in time window

#### Story 4: Configurable Time Window (tmux-intray-n99)

Add `recents_time_window` config option

- Default value: "1h"
- Validate config at startup
- Document in configuration.md

### Non-Functional Requirements

- No grouping/collapsible sections (keep flat list)
- No changes to All tab behavior
- No changes to CLI `status --format` presets
- Performance should not degrade with many sessions
- Time to load filtered list should not increase

## Design Decisions to Document

### Decision 1: Time Window Default = 1 hour

**Rationale:** User mentioned "last 30 minutes, 1 hour", 1h balances "recent" with "not too strict", configurable

**Trade-offs:**

- Matches user expectation ✅
- Configurable ✅
- Might be too long/short for some users ⚠️

### Decision 2: Per-Session Limit = 1 max

**Rationale:** User wants diversity across sessions, prevents noisy sessions from dominating, keeps list compact

**Trade-offs:**

- Shows more sessions ✅
- Quick scan ✅
- Compact ✅
- Might miss context ⚠️

### Decision 3: Selection Strategy = Severity Priority

**Algorithm:** For each session: if error → most recent error, elif warning → most recent warning, else → most recent

**Rationale:** Errors are most important, don't show info if error exists

**Trade-offs:**

- Shows most important ✅
- Not overwhelming ✅
- Might miss recent info if error old ⚠️

### Decision 4: Filtered List Limit = 10 max

**Rationale:** User wants context before jumping, 10 is enough without overwhelming, respects time window

**Trade-offs:**

- Shows recent activity ✅
- Respects time window ✅
- Limit arbitrary ⚠️

### Decision 5: Configurable Time Window = Predefined Options

**Options:** "5m", "15m", "30m", "1h", "2h", "6h", "12h", "24h"

**Rationale:** Different workflows need different definitions, predefined values prevent invalid configs

**Trade-offs:**

- Flexibility ✅
- Simple validation ✅
- Limited options ⚠️

## Implementation Plan

**Phase 1 (Story 1):** Add time window filtering to Recents query, add config option, add empty state message

**Phase 2 (Story 2):** Implement per-session limiting, severity-based selection, update ordering logic

**Phase 3 (Story 3):** Clarify filtered list behavior, add limit for filtered lists

**Phase 4 (Story 4 + All):** Document config option, add/update tests, user acceptance testing

## Success Criteria

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Time to find relevant notification after sound | < 5 seconds | Manual testing / telemetry |
| User satisfaction with "recent" definition | High | Qualitative feedback |
| Sessions visible in Recents (unfiltered) | Min 3, max 20 | Observation |
| Cognitive load | No increase | Qualitative feedback |

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| 1-hour window too long | Medium | Low | Configurable, can adjust down |
| 1 item per session misses context | Medium | Medium | Filtered list shows up to 10 |
| Performance degrades | High | Low | Benchmark with 100+ sessions |
| Users confused by empty Recents | Medium | Low | Clear message explaining why |

## Example Scenarios to Document

### Scenario A: Sound-Driven Triage

```
User hears sound → opens Recents

Current:
  [INFO] Session A: Build started (1h ago)
  [INFO] Session A: Tests running (1h ago)
  [ERROR] Session A: Build failed (55m ago)
  [INFO] Session B: Deploy done (2h ago)

New (1h window, 1 per session):
  [ERROR] Session A: Build failed (55m ago)
  [INFO] Session B: Deploy done (2h ago) - NOT SHOWN (> 1h)
```

### Scenario B: Noisy Session

```
5 sessions active in last hour:
- Session A: 50 notifications
- Session B: 10 notifications
- Session C: 5 notifications
- Session D: 2 notifications
- Session E: 1 notification

Current (3 per source): Shows 14 items, Sessions A and B dominate
New (1 per session): Shows 5 items, all sessions represented with severity awareness
```

### Scenario C: Filtered List

```
User filters "session-a" (50 notifications in last hour)

Current: Shows up to 20
New: Shows up to 10 (respect time window, ordered newest first)
```

## Current Implementation Notes

From `internal/tui/service/notification_service.go`:

```go
recentsDatasetLimit   = 20  // Latest 20 overall
recentsPerSourceLimit = 3   // Max 3 per source

// Issues:
// 1. Time-based filtering missing (can show day-old items)
// 2. Per-source limit allows noisy sessions to dominate
// 3. No severity awareness in selection
```

## Related Beads Issues

- Epic: `tmux-intray-3wm` - Improve Recents Tab
- Story 1: `tmux-intray-nkv` - Time-Based Filtering
- Story 2: `tmux-intray-lkb` - Per-Session Smart Selection
- Story 3: `tmux-intray-2c4` - Filtered List Behavior
- Story 4: `tmux-intray-n99` - Configurable Time Window

## Document History

| Date | Author | Change |
|------|--------|--------|
| 2026-03-15 | cristianoliveira | Initial design brief based on user interview |
| 2026-03-15 | cristianoliveira | Enhanced with detailed problem statement and user workflow context |
