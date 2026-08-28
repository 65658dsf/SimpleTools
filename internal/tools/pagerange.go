package tools

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ParsePageRange accepts expressions such as "1-3,5,8-". Pages are one-based.
func ParsePageRange(expr string, pageCount int) ([]int, error) {
	if pageCount < 1 {
		return nil, fmt.Errorf("page count must be positive")
	}
	if strings.TrimSpace(expr) == "" {
		out := make([]int, pageCount)
		for i := range out {
			out[i] = i + 1
		}
		return out, nil
	}
	seen := map[int]bool{}
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty page range")
		}
		bounds := strings.Split(part, "-")
		if len(bounds) > 2 {
			return nil, fmt.Errorf("invalid page range %q", part)
		}
		parse := func(s string) (int, error) {
			n, e := strconv.Atoi(strings.TrimSpace(s))
			if e != nil || n < 1 {
				return 0, fmt.Errorf("invalid page number %q", s)
			}
			return n, nil
		}
		start, e := parse(bounds[0])
		if e != nil {
			return nil, e
		}
		end := start
		if len(bounds) == 2 {
			if strings.TrimSpace(bounds[1]) == "" {
				end = pageCount
			} else {
				end, e = parse(bounds[1])
				if e != nil {
					return nil, e
				}
			}
		}
		if start > end || end > pageCount {
			return nil, fmt.Errorf("page range %q is outside 1-%d", part, pageCount)
		}
		for i := start; i <= end; i++ {
			seen[i] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}
