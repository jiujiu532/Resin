# Node Management V2 - Design Spec

## Overview

Evolve the current platform-level blocklist into a full node lifecycle management system with:
- Global node pool: disable / enable / delete with cascading effects
- Platform-level node pool: disable / enable (local scope) + delete (global cascade)
- Platform subscription source selector (pill-style multi-select)
- Batch operations with checkbox selection

---

## 1. Concepts and Hierarchy

```
Global Node Pool (all nodes from all subscriptions)
  |
  +-- Platform A (filtered subset: subscription sources + regex + region + blocklist)
  |     |-- enabled nodes (participate in routing)
  |     +-- disabled nodes (exist but excluded from routing)
  |
  +-- Platform B (different filtered subset)
        |-- enabled nodes
        +-- disabled nodes
```

**Key rules:**
- Global Pool > Platform Pool (platforms are subsets)
- Global disable/enable/delete cascades DOWN to all platforms containing that node
- Platform disable/enable is LOCAL (only affects that platform)
- Delete is ALWAYS GLOBAL regardless of where triggered

---

## 2. Node States

Each node has two independent state dimensions:

| Dimension | Scope | Values |
|-----------|-------|--------|
| Global enabled | Global pool | enabled / disabled |
| Platform enabled | Per-platform | enabled / disabled (via blockedNodes) |

A node is routable in a platform only when: `global_enabled AND platform_enabled AND passes_filters AND healthy`

---

## 3. Feature Breakdown

### 3.1 Global Node Pool Page (rename: "Total Node Pool")

**UI changes:**
- Page title: "Total Node Pool" (was "Node Pool")
- Top toolbar: add 3 action buttons: [Disable] [Enable] [Delete]
- Each row: add checkbox on the left
- Table header: add "Select All" checkbox
- Buttons are disabled when no nodes are selected

**Operations:**
- **Disable (batch):** Set selected nodes as globally disabled. All platforms containing these nodes will also see them as disabled. Persisted to `disabled_node_hashes_json` in a new global config field.
- **Enable (batch):** Remove selected nodes from global disabled list. Platforms that had them platform-disabled remain platform-disabled.
- **Delete (batch):** Permanently remove nodes from the global pool. Shows danger confirmation dialog with "I understand the risk, don't remind me again" checkbox. Removes from all subscriptions' managed nodes, all platforms' blocklists, and the global pool.

**API endpoints:**
- `POST /api/v1/nodes/actions/disable` body: `{"node_hashes": [...]}`
- `POST /api/v1/nodes/actions/enable` body: `{"node_hashes": [...]}`
- `POST /api/v1/nodes/actions/delete` body: `{"node_hashes": [...]}`

### 3.2 Platform Subscription Source Selector

**Current behavior:** Platform uses regex_filters + region_filters to auto-select nodes from the entire global pool.

**New behavior (additive):** Platform gains a `subscription_sources` field (list of subscription IDs). When non-empty, only nodes from those subscriptions are candidates for the platform's routable view. Regex and region filters still apply as secondary filters on top.

**UI changes (Platform Detail > Config tab):**
- Replace the current regex_filters textarea label area with a new section: "Subscription Sources"
- Dropdown to select from all available subscriptions
- Selected subscriptions shown as pill/tag chips above the dropdown
- Each pill has an X button to remove
- Empty selection = use all subscriptions (backward compatible)
- Keep regex_filters and region_filters below as secondary filters

**Data model:**
- `model.Platform` adds `SubscriptionSources []string` (subscription IDs)
- DB migration: `ALTER TABLE platforms ADD COLUMN subscription_sources_json TEXT NOT NULL DEFAULT '[]'`
- `evaluateNode` adds check: if SubscriptionSources is non-empty, node must belong to at least one of those subscriptions

### 3.3 Platform Node Pool

**UI changes (Platform Detail > Monitor tab or new "Nodes" tab):**
- "Routable Nodes" link renamed to "{PlatformName} Node Pool"
- Shows: total nodes in platform, enabled count, disabled count
- Node list with same checkbox + batch operations as global pool
- Top toolbar: [Disable] [Enable] [Delete]

**Operations:**
- **Disable (platform-local):** Add to platform's `blocked_node_hashes`. Only affects this platform.
- **Enable (platform-local):** Remove from platform's `blocked_node_hashes`. Only affects this platform.
- **Delete (global cascade):** Same as global delete. Shows danger dialog. Removes from ALL platforms and global pool.

**API endpoints (reuse existing + new):**
- Platform disable = existing `POST /platforms/{id}/blocked-nodes` (block)
- Platform enable = existing `DELETE /platforms/{id}/blocked-nodes/{hash}` (unblock)
- Delete = same global delete endpoint `POST /api/v1/nodes/actions/delete`

### 3.4 Delete Confirmation Dialog

**Behavior:**
- Modal dialog with danger styling
- Text: "Deleting nodes is irreversible. Selected nodes will be permanently removed from all platforms and the global pool."
- Shows count of nodes to be deleted
- Bottom-right: checkbox "I understand the risk, don't remind me again"
- Buttons: [Cancel] [Confirm Delete]
- "Don't remind" preference stored in localStorage

---

## 4. Data Model Changes

### 4.1 New: Global Disabled Nodes

Store globally disabled node hashes. Options:
- **Option A (recommended):** New table `disabled_nodes(node_hash TEXT PRIMARY KEY)` in state.db
- **Option B:** JSON field in system_config

Choose Option A for better query performance with large lists.

**DB migration 000007:**
```sql
CREATE TABLE IF NOT EXISTS disabled_nodes (
    node_hash TEXT PRIMARY KEY
);
```

### 4.2 Platform Model Update

Add `SubscriptionSources []string` to `model.Platform`.

**DB migration 000008:**
```sql
ALTER TABLE platforms ADD COLUMN subscription_sources_json TEXT NOT NULL DEFAULT '[]';
```

### 4.3 evaluateNode Update

```
func evaluateNode(entry, subLookup, geoLookup) bool:
    0. if globally disabled -> false
    1. if disabled by subscriptions -> false
    2. if not healthy -> false
    3. if subscription_sources non-empty AND node not in any source -> false
    4. if regex filter fails -> false
    5. if egress IP unknown -> false
    6. if region filter fails -> false
    7. if no latency -> false
    8. if platform-blocked -> false
    return true
```

---

## 5. API Summary

| Method | Path | Scope | Description |
|--------|------|-------|-------------|
| POST | /api/v1/nodes/actions/disable | Global | Batch disable nodes globally |
| POST | /api/v1/nodes/actions/enable | Global | Batch enable nodes globally |
| POST | /api/v1/nodes/actions/delete | Global | Batch delete nodes permanently |
| GET | /api/v1/nodes | Global | List nodes (add `globally_disabled` filter) |
| POST | /api/v1/platforms/{id}/blocked-nodes | Platform | Disable node in platform (existing) |
| DELETE | /api/v1/platforms/{id}/blocked-nodes/{hash} | Platform | Enable node in platform (existing) |
| POST | /api/v1/platforms/{id}/blocked-nodes/batch | Platform | Batch disable in platform |
| DELETE | /api/v1/platforms/{id}/blocked-nodes/batch | Platform | Batch enable in platform |

---

## 6. Frontend Changes Summary

| Page | Change |
|------|--------|
| Node Pool page | Rename to "Total Node Pool", add checkboxes + select-all + batch toolbar |
| Platform Detail > Config | Add subscription source pill selector above regex/region filters |
| Platform Detail > Monitor | Rename "Routable Nodes" display, show enabled/disabled counts |
| Platform Node Pool view | Add checkboxes + batch toolbar (disable/enable/delete) |
| Delete dialog | New reusable modal component with "don't remind" localStorage |

---

## 7. Implementation Order (Sub-projects)

1. **Sub-project 1:** Global disable/enable + delete API + DB migration + evaluateNode update
2. **Sub-project 2:** Frontend batch operations (checkboxes, toolbar, delete dialog) for global pool
3. **Sub-project 3:** Platform subscription source selector (backend + frontend)
4. **Sub-project 4:** Platform node pool view with local disable/enable + batch operations

Each sub-project is independently deployable and testable.

---

## 8. Backward Compatibility

- Empty `subscription_sources` = all subscriptions (current behavior preserved)
- Empty `disabled_nodes` table = all nodes enabled (current behavior preserved)
- Existing `blocked_node_hashes` continues to work as platform-level disable
- No breaking API changes; all new endpoints are additive
