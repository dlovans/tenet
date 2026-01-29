package tenet

import (
	"fmt"
	"time"
)

// validateTemporalMap checks for configuration errors in temporal_map.
// Detects same start/end dates and overlapping ranges.
func (e *Engine) validateTemporalMap() {
	if e.schema.TemporalMap == nil || len(e.schema.TemporalMap) == 0 {
		return
	}

	for i, branch := range e.schema.TemporalMap {
		if branch == nil {
			continue
		}

		start := branch.ValidRange[0]
		end := branch.ValidRange[1]

		// Check for same start/end date (invalid zero-length range)
		if start != nil && end != nil && *start == *end {
			e.addError("", "", fmt.Sprintf(
				"Temporal branch %d has same start and end date '%s' (invalid range)",
				i, *start), "")
		}

		// Check for overlapping with previous branch
		if i > 0 {
			prev := e.schema.TemporalMap[i-1]
			if prev != nil {
				var prevEndTime int64 = 1<<62 - 1 // Max int64 (infinity)
				if prev.ValidRange[1] != nil {
					if parsed, ok := parseDate(*prev.ValidRange[1]); ok {
						prevEndTime = parsed.Unix()
					}
				}

				var currStartTime int64 = -(1<<62 - 1) // Min int64 (-infinity)
				if start != nil {
					if parsed, ok := parseDate(*start); ok {
						currStartTime = parsed.Unix()
					}
				}

				if currStartTime <= prevEndTime {
					e.addError("", "", fmt.Sprintf(
						"Temporal branch %d overlaps with branch %d (ranges must not overlap)",
						i, i-1), "")
				}
			}
		}
	}
}

// selectBranch finds the active temporal branch for a given effective date.
// Returns nil if no branch matches (uses default/unversioned logic).
func (e *Engine) selectBranch(targetDate time.Time) *TemporalBranch {
	if e.schema.TemporalMap == nil {
		return nil
	}

	for _, branch := range e.schema.TemporalMap {
		if branch == nil || branch.ValidRange[0] == nil {
			continue
		}

		start, ok := parseDate(*branch.ValidRange[0])
		if !ok {
			continue
		}

		// Check if targetDate is at or after start
		if targetDate.Before(start) {
			continue
		}

		// Check end date (nil = open-ended)
		if branch.ValidRange[1] != nil {
			end, ok := parseDate(*branch.ValidRange[1])
			if ok && targetDate.After(end) {
				continue
			}
		}

		return branch
	}

	return nil
}

// prune marks rules as disabled if they don't belong to the active branch.
// Rules without a logic_version are always active (unversioned rules).
func (e *Engine) prune(activeBranch *TemporalBranch) {
	if activeBranch == nil {
		return
	}

	activeVersion := activeBranch.LogicVersion

	for _, rule := range e.schema.LogicTree {
		if rule == nil {
			continue
		}

		// Rules without a version are always active
		if rule.LogicVersion == "" {
			continue
		}

		// Disable rules that don't match the active version
		if rule.LogicVersion != activeVersion {
			rule.Disabled = true
		}
	}
}

// getActiveVersion returns the logic version for a given date.
// Returns empty string if no temporal mapping exists.
func (e *Engine) getActiveVersion(targetDate time.Time) string {
	branch := e.selectBranch(targetDate)
	if branch == nil {
		return ""
	}
	return branch.LogicVersion
}
