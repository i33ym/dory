// Package filter provides MetadataFilter translation utilities used
// internally by VectorStore implementations. It is not part of the public API.
package filter

import (
	"fmt"

	"github.com/i33ym/dory"
)

// Match checks whether a metadata map satisfies a single filter.
func Match(metadata map[string]any, f *dory.MetadataFilter) bool {
	if metadata == nil || f == nil {
		return false
	}
	val, ok := metadata[f.Field]
	if !ok {
		return false
	}

	switch f.Op {
	case dory.FilterOpEq:
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", f.Value)

	case dory.FilterOpIn:
		// value is a list; field value must equal one of the items.
		list := toStringSlice(f.Value)
		s := fmt.Sprintf("%v", val)
		for _, item := range list {
			if s == item {
				return true
			}
		}
		return false

	case dory.FilterOpAnyOf:
		// field value is a list; at least one element must appear in the filter's value list.
		filterSet := toStringSlice(f.Value)
		fieldList := toStringSlice(val)
		for _, fv := range fieldList {
			for _, sv := range filterSet {
				if fv == sv {
					return true
				}
			}
		}
		return false

	default:
		return false
	}
}

// MatchAll returns true only if every filter matches.
func MatchAll(metadata map[string]any, filters []dory.MetadataFilter) bool {
	for i := range filters {
		if !Match(metadata, &filters[i]) {
			return false
		}
	}
	return true
}

// toStringSlice converts various slice types to []string for comparison.
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, len(s))
		for i, item := range s {
			out[i] = fmt.Sprintf("%v", item)
		}
		return out
	default:
		return nil
	}
}
