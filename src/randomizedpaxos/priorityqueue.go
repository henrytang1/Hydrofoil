// This example demonstrates a priority queue built using the heap interface.
package randomizedpaxos

import (
	"container/heap"
	"sort"
	"state"
)

type UniqueCommand struct {
	senderId 	int32
	time	 	int64
}

type ExtendedPriorityQueue struct {
	pq 		PriorityQueue
	itemLoc		map[UniqueCommand]*Item
}

func newExtendedPriorityQueue() ExtendedPriorityQueue {
	var extPQ ExtendedPriorityQueue
	extPQ.pq = make(PriorityQueue, 0)
	extPQ.itemLoc = make(map[UniqueCommand]*Item, 0)
	heap.Init(&extPQ.pq)
	return extPQ
}

func (extPQ *ExtendedPriorityQueue) push(entry Entry) {
	if entry.Term == -1 {
		return
	}

	entry.Term = -1
	entry.Index = -1
	item := &Item{
		entry: entry,
		heapIndex: -1, // initial heapIndex doesn't matter
	}

	req := UniqueCommand {
		senderId: entry.SenderId,
		time: entry.Timestamp,
	}
	
	if _, ok := extPQ.itemLoc[req]; ok {
		return		
	}
	heap.Push(&extPQ.pq, item)
	extPQ.itemLoc[req] = item
}

func (extPQ *ExtendedPriorityQueue) contains(entry Entry) bool {
	req := UniqueCommand {
		senderId: entry.SenderId,
		time: entry.Timestamp,
	}

	if _, ok := extPQ.itemLoc[req]; ok {
		return true
	}
	return false
}

func (extPQ *ExtendedPriorityQueue) remove(entry Entry) {
	req := UniqueCommand {
		senderId: entry.SenderId,
		time: entry.Timestamp,
	}

	if val, ok := extPQ.itemLoc[req]; ok {
		heap.Remove(&extPQ.pq, val.heapIndex)
		delete(extPQ.itemLoc, req)
	}
}

func (extPQ *ExtendedPriorityQueue) pop() Entry {
	item := heap.Pop(&extPQ.pq).(*Item)

	req := UniqueCommand {
		senderId: item.entry.SenderId,
		time: item.entry.Timestamp,
	}

	delete(extPQ.itemLoc, req)
	return item.entry
}

func (extPQ *ExtendedPriorityQueue) peek() *Item {
	return extPQ.pq[0]
}

// func (extPQ *ExtendedPriorityQueue) clearLeaderEntries(){
// 	for len(extPQ.pq) > 0 && extPQ.peek().entry.Term != -1 {
// 		extPQ.pop()
// 	}
// }

func (extPQ *ExtendedPriorityQueue) popAll() []Entry {
	var list []Entry
	for i := 0; i < len(extPQ.pq); i++ {
		list = append(list, extPQ.pq[i].entry)
	}
	sort.Slice(list, func(i, j int) bool {
		return cmpEntry(list[i], list[j])
	})
	return list
}

func (extPQ *ExtendedPriorityQueue) extractList() []Entry {
	list := extPQ.popAll()
	for _, v := range list {
		extPQ.push(v)
	}
	return list
}

func (extPQ *ExtendedPriorityQueue) isEmpty() bool {
	return len(extPQ.pq) == 0
}

// An Item is something we manage in a priority queue.
type Item struct {
	entry Entry
	heapIndex int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func cmpEntry(a, b Entry) bool {
	if a.Timestamp != b.Timestamp {
		return a.Timestamp < b.Timestamp
	}
	return a.SenderId < b.SenderId
}

func cmpItem(a, b *Item) bool {
	return cmpEntry(a.entry, b.entry)
}

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return cmpItem(pq[i], pq[j])
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].heapIndex = i
	pq[j].heapIndex = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.heapIndex = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.heapIndex = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
// this function might not be necessary
func (pq *PriorityQueue) update(item *Item, entry Entry, request state.Command) {
	item.entry = entry
	heap.Fix(pq, item.heapIndex)
}

func equalEntry(a, b Entry) bool {
	return a.SenderId == b.SenderId && a.Timestamp == b.Timestamp
}

// // This example creates a PriorityQueue with some items, adds and manipulates an item,
// // and then removes the items in priority order.
// func main() {
// 	// Some items and their priorities.
// 	items := map[string]int{
// 		"banana": 3, "apple": 2, "pear": 4,
// 	}

// 	// Create a priority queue, put the items in it, and
// 	// establish the priority queue (heap) invariants.
// 	pq := make(PriorityQueue, len(items))
// 	i := 0
// 	for value, priority := range items {
// 		pq[i] = &Item{
// 			value:    value,
// 			priority: priority,
// 			heapIndex:    i,
// 		}
// 		i++
// 	}
// 	heap.Init(&pq)

// 	// Insert a new item and then modify its priority.
// 	item := &Item{
// 		value:    "orange",
// 		priority: 1,
// 	}
// 	heap.Push(&pq, item)
// 	pq.update(item, item.value, 5)

// 	// Take the items out; they arrive in decreasing priority order.
// 	for pq.Len() > 0 {
// 		item := heap.Pop(&pq).(*Item)
// 		fmt.Printf("%.2d:%s ", item.priority, item.value)
// 	}
// }
