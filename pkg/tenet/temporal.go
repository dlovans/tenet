package tenet

import (
	"time"
)

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
