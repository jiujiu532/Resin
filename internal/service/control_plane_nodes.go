package service

import (
	"strings"
	"time"

	"github.com/jiujiu532/Resin/internal/node"
	"github.com/jiujiu532/Resin/internal/probe"
	"github.com/jiujiu532/Resin/internal/subscription"
)

// ------------------------------------------------------------------
// Nodes
// ------------------------------------------------------------------

// NodeFilters holds query filters for listing nodes.
type NodeFilters struct {
	PlatformID     *string
	SubscriptionID *string
	Enabled        *bool
	Region         *string
	CircuitOpen    *bool
	HasOutbound    *bool
	EgressIP       *string
	ProbedSince    *time.Time
	TagKeyword     *string
}

// ListNodes returns nodes from the pool with optional filters.
func (s *ControlPlaneService) ListNodes(filters NodeFilters) ([]NodeSummary, error) {
	var subLookup node.SubLookupFunc
	if s != nil && s.Pool != nil {
		subLookup = s.Pool.MakeSubLookup()
	}

	// If platform_id filter, get the platform view.
	var platformView map[node.Hash]struct{}
	if filters.PlatformID != nil {
		plat, ok := s.Pool.GetPlatform(*filters.PlatformID)
		if !ok {
			return nil, notFound("platform not found")
		}
		platformView = make(map[node.Hash]struct{}, plat.View().Size())
		plat.View().Range(func(h node.Hash) bool {
			platformView[h] = struct{}{}
			return true
		})
	}

	var subNodes map[node.Hash]struct{}
	if filters.SubscriptionID != nil {
		sub := s.SubMgr.Lookup(*filters.SubscriptionID)
		if sub == nil {
			return nil, notFound("subscription not found")
		}
		subNodes = make(map[node.Hash]struct{})
		sub.ManagedNodes().RangeNodes(func(h node.Hash, managed subscription.ManagedNode) bool {
			if managed.Evicted {
				return true
			}
			subNodes[h] = struct{}{}
			return true
		})
	}

	var result []NodeSummary
	appendIfMatched := func(h node.Hash, entry *node.NodeEntry) {
		if !s.nodeEntryMatchesFilters(entry, filters, subLookup) {
			return
		}
		result = append(result, s.nodeEntryToSummary(h, entry))
	}

	appendIfMatchedHash := func(h node.Hash) {
		entry, ok := s.Pool.GetEntry(h)
		if !ok {
			return
		}
		appendIfMatched(h, entry)
	}

	switch {
	case platformView != nil && subNodes != nil:
		// Iterate the smaller candidate set, then intersect by membership.
		if len(platformView) <= len(subNodes) {
			for h := range platformView {
				if _, ok := subNodes[h]; !ok {
					continue
				}
				appendIfMatchedHash(h)
			}
		} else {
			for h := range subNodes {
				if _, ok := platformView[h]; !ok {
					continue
				}
				appendIfMatchedHash(h)
			}
		}
	case platformView != nil:
		for h := range platformView {
			appendIfMatchedHash(h)
		}
	case subNodes != nil:
		for h := range subNodes {
			appendIfMatchedHash(h)
		}
	default:
		s.Pool.Range(func(h node.Hash, entry *node.NodeEntry) bool {
			appendIfMatched(h, entry)
			return true
		})
	}

	if result == nil {
		result = []NodeSummary{}
	}
	return result, nil
}

func (s *ControlPlaneService) nodeEntryMatchesFilters(
	entry *node.NodeEntry,
	filters NodeFilters,
	subLookup node.SubLookupFunc,
) bool {
	// Enabled/disabled filter.
	if filters.Enabled != nil {
		enabled := true
		if subLookup != nil {
			enabled = entry.HasEnabledSubscription(subLookup)
		}
		if enabled != *filters.Enabled {
			return false
		}
	}

	// Node tag fuzzy search filter.
	if filters.TagKeyword != nil {
		keyword := strings.ToLower(strings.TrimSpace(*filters.TagKeyword))
		if keyword != "" {
			matched := false
			for _, subID := range entry.SubscriptionIDs() {
				sub := s.SubMgr.Lookup(subID)
				if sub == nil {
					continue
				}
				managed, ok := sub.ManagedNodes().LoadNode(entry.Hash)
				if !ok {
					continue
				}
				tags := managed.Tags
				for _, tag := range tags {
					displayTag := sub.Name() + "/" + tag
					if strings.Contains(strings.ToLower(displayTag), keyword) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	// Region filter.
	if filters.Region != nil {
		region := entry.GetRegion(nil)
		if s.GeoIP != nil {
			region = entry.GetRegion(s.GeoIP.Lookup)
		}
		if region == "" || region != *filters.Region {
			return false
		}
	}
	// Circuit open filter.
	if filters.CircuitOpen != nil {
		if entry.IsCircuitOpen() != *filters.CircuitOpen {
			return false
		}
	}
	// Has outbound filter.
	if filters.HasOutbound != nil {
		if entry.HasOutbound() != *filters.HasOutbound {
			return false
		}
	}
	// Egress IP filter.
	if filters.EgressIP != nil {
		egressIP := entry.GetEgressIP()
		if !egressIP.IsValid() || egressIP.String() != *filters.EgressIP {
			return false
		}
	}
	// Probed since filter.
	if filters.ProbedSince != nil {
		lastUpdate := entry.LastLatencyProbeAttempt.Load()
		if lastUpdate < filters.ProbedSince.UnixNano() {
			return false
		}
	}
	return true
}

// GetNode returns a single node by hash.
func (s *ControlPlaneService) GetNode(hashStr string) (*NodeSummary, error) {
	h, err := node.ParseHex(hashStr)
	if err != nil {
		return nil, invalidArg("node_hash: invalid format")
	}
	entry, ok := s.Pool.GetEntry(h)
	if !ok {
		return nil, notFound("node not found")
	}
	ns := s.nodeEntryToSummary(h, entry)
	return &ns, nil
}

// ProbeEgress triggers a synchronous egress probe and returns results.
func (s *ControlPlaneService) ProbeEgress(hashStr string) (*probe.EgressProbeResult, error) {
	h, err := node.ParseHex(hashStr)
	if err != nil {
		return nil, invalidArg("node_hash: invalid format")
	}
	entry, ok := s.Pool.GetEntry(h)
	if !ok {
		return nil, notFound("node not found")
	}
	result, err := s.ProbeMgr.ProbeEgressSync(h)
	if err != nil {
		return nil, internal("egress probe failed", err)
	}
	result.Region = entry.GetRegion(nil)
	if s.GeoIP != nil {
		result.Region = entry.GetRegion(s.GeoIP.Lookup)
	}
	return result, nil
}

// ProbeLatency triggers a synchronous latency probe and returns results.
func (s *ControlPlaneService) ProbeLatency(hashStr string) (*probe.LatencyProbeResult, error) {
	h, err := node.ParseHex(hashStr)
	if err != nil {
		return nil, invalidArg("node_hash: invalid format")
	}
	if _, ok := s.Pool.GetEntry(h); !ok {
		return nil, notFound("node not found")
	}
	result, err := s.ProbeMgr.ProbeLatencySync(h)
	if err != nil {
		return nil, internal("latency probe failed", err)
	}
	return result, nil
}

// ------------------------------------------------------------------
// Global node disable / enable / delete
// ------------------------------------------------------------------

// DisableNodesGlobally marks nodes as globally disabled and persists the change.
func (s *ControlPlaneService) DisableNodesGlobally(nodeHashes []string) error {
	if len(nodeHashes) == 0 {
		return nil
	}
	hashes := make([]node.Hash, 0, len(nodeHashes))
	for _, hexStr := range nodeHashes {
		h, err := node.ParseHex(hexStr)
		if err != nil {
			return invalidArg("node_hash: invalid hex format: " + hexStr)
		}
		hashes = append(hashes, h)
	}
	if err := s.Engine.AddDisabledNodes(nodeHashes); err != nil {
		return internal("persist disabled nodes", err)
	}
	s.Pool.DisableNodes(hashes)
	return nil
}

// EnableNodesGlobally removes nodes from the global disabled set and persists the change.
func (s *ControlPlaneService) EnableNodesGlobally(nodeHashes []string) error {
	if len(nodeHashes) == 0 {
		return nil
	}
	hashes := make([]node.Hash, 0, len(nodeHashes))
	for _, hexStr := range nodeHashes {
		h, err := node.ParseHex(hexStr)
		if err != nil {
			return invalidArg("node_hash: invalid hex format: " + hexStr)
		}
		hashes = append(hashes, h)
	}
	if err := s.Engine.RemoveDisabledNodes(nodeHashes); err != nil {
		return internal("persist enabled nodes", err)
	}
	s.Pool.EnableNodes(hashes)
	return nil
}

// DeleteNodesGlobally permanently removes nodes from the pool, all subscriptions,
// all platform blocklists, and the global disabled set.
func (s *ControlPlaneService) DeleteNodesGlobally(nodeHashes []string) error {
	if len(nodeHashes) == 0 {
		return nil
	}
	hashes := make([]node.Hash, 0, len(nodeHashes))
	for _, hexStr := range nodeHashes {
		h, err := node.ParseHex(hexStr)
		if err != nil {
			return invalidArg("node_hash: invalid hex format: " + hexStr)
		}
		hashes = append(hashes, h)
	}

	// Remove from pool (cascades to subscriptions).
	s.Pool.DeleteNodes(hashes)

	// Remove from all platform blocklists in memory.
	for _, h := range hashes {
		s.Pool.RemoveNodeFromAllPlatformBlocklists(h)
	}

	// Remove from global disabled set.
	if err := s.Engine.RemoveDisabledNodes(nodeHashes); err != nil {
		return internal("remove from disabled nodes", err)
	}

	// Remove from all platform blocked_node_hashes in DB.
	platforms, err := s.Engine.ListPlatforms()
	if err != nil {
		return internal("list platforms for blocklist cleanup", err)
	}
	hashSet := make(map[string]struct{}, len(nodeHashes))
	for _, h := range nodeHashes {
		hashSet[h] = struct{}{}
	}
	for _, mp := range platforms {
		changed := false
		newHashes := mp.BlockedNodeHashes[:0:0]
		for _, h := range mp.BlockedNodeHashes {
			if _, del := hashSet[h]; del {
				changed = true
				continue
			}
			newHashes = append(newHashes, h)
		}
		if changed {
			mp.BlockedNodeHashes = newHashes
			mp.UpdatedAtNs = time.Now().UnixNano()
			if err := s.Engine.UpsertPlatform(mp); err != nil {
				return internal("update platform blocklist after delete", err)
			}
		}
	}

	return nil
}
