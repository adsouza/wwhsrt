package domain

import "sort"

type weighted struct {
	id     string
	weight int64
}

// Largest remainders conserve the finite market exactly. IDs settle rounding
// ties, never map iteration or service submission order.
func shares(total int, weights []weighted) []int {
	out := make([]int, len(weights))
	var sum int64
	for _, w := range weights {
		sum += w.weight
	}
	if sum == 0 {
		return out
	}
	order := make([]int, len(weights))
	used := 0
	for i, w := range weights {
		out[i] = int(int64(total) * w.weight / sum)
		used += out[i]
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		a, b := order[i], order[j]
		x, y := int64(total)*weights[a].weight%sum, int64(total)*weights[b].weight%sum
		if x != y {
			return x > y
		}
		return weights[a].id < weights[b].id
	})
	for i := 0; i < total-used; i++ {
		out[order[i]]++
	}
	return out
}

type capacityRequest struct {
	id     string
	need   int
	legs   []string
	served int
}

// Allocate complete itineraries, progressively releasing uncongested requests
// from bottleneck groups. Every reservation consumes all its legs atomically.
// The same routine allocates train paths and passenger seats at different scales.
func ration(requests []capacityRequest, remaining map[string]int) {
	for {
		active := []int{}
		totals := map[string]int{}
		for i, r := range requests {
			if r.served >= r.need {
				continue
			}
			possible := true
			for _, leg := range r.legs {
				if remaining[leg] <= 0 {
					possible = false
					break
				}
			}
			if !possible {
				continue
			}
			active = append(active, i)
			for _, leg := range r.legs {
				totals[leg] += r.need - r.served
			}
		}
		if len(active) == 0 {
			return
		}
		num, den := int64(1), int64(1)
		for leg, total := range totals {
			if int64(remaining[leg])*den < num*int64(total) {
				num, den = int64(remaining[leg]), int64(total)
			}
		}
		progress := false
		add := func(i, n int) {
			if n <= 0 {
				return
			}
			requests[i].served += n
			for _, leg := range requests[i].legs {
				remaining[leg] -= n
			}
			progress = true
		}
		residual := map[int]int64{}
		for _, i := range active {
			left := requests[i].need - requests[i].served
			residual[i] = int64(left) * num % den
			add(i, int(int64(left)*num/den))
		}
		sort.Slice(active, func(i, j int) bool {
			a, b := active[i], active[j]
			if residual[a] != residual[b] {
				return residual[a] > residual[b]
			}
			return requests[a].id < requests[b].id
		})
		for _, i := range active {
			if residual[i] == 0 || requests[i].served >= requests[i].need {
				continue
			}
			possible := true
			for _, leg := range requests[i].legs {
				if remaining[leg] <= 0 {
					possible = false
					break
				}
			}
			if possible {
				add(i, 1)
			}
		}
		if !progress {
			return
		}
	}
}
