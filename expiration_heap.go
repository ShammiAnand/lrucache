package lrucache

import "time"

type expirationHeap struct {
	items     []string
	expiresAt map[string]time.Time
}

func (h *expirationHeap) Len() int { return len(h.items) }
func (h *expirationHeap) Less(i, j int) bool {
	return h.expiresAt[h.items[i]].Before(h.expiresAt[h.items[j]])
}
func (h *expirationHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}
func (h *expirationHeap) Push(x interface{}) {
	h.items = append(h.items, x.(string))
}
func (h *expirationHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[0 : n-1]
	return x
}
