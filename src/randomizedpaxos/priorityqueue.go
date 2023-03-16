// This example demonstrates a priority queue built using the heap interface.
package randomizedpaxos

import (
	"container/heap"
	"randomizedpaxosproto"
	"sort"
	"state"
)

type UniqueCommand struct {
	senderId int32
	term 	 int32
	index	 int32
	time	 int64
}

type ExtendedPriorityQueue struct {
	pq 			PriorityQueue
	itemLoc		map[UniqueCommand]*Item
}

func (extPQ ExtendedPriorityQueue) push(entry randomizedpaxosproto.Entry) {
	item := &Item{
		// senderId: entry.SenderId,
		// term: entry.Term,
		// index: entry.Index,
		// time: entry.Timestamp,
		// request: entry.Data,
		entry: entry,
		heapIndex: -1, // initial heapIndex doesn't matter
	}

	req := UniqueCommand {
		senderId: entry.SenderId,
		term: entry.Term,
		index: entry.Index,
		time: entry.Timestamp,
	}
	
	if val, ok := extPQ.itemLoc[req]; ok {
		if cmpItem(item, val) {
			heap.Remove(&extPQ.pq, val.heapIndex)
			extPQ.pq.Push(item)
			extPQ.itemLoc[req] = item
		}
	}
}

func (extPQ ExtendedPriorityQueue) remove(entry randomizedpaxosproto.Entry) {
	req := UniqueCommand {
		senderId: entry.SenderId,
		term: entry.Term,
		index: entry.Index,
		time: entry.Timestamp,
	}

	if val, ok := extPQ.itemLoc[req]; ok {
		heap.Remove(&extPQ.pq, val.heapIndex)
		delete(extPQ.itemLoc, req)
	}
}

func (extPQ ExtendedPriorityQueue) pop() any {
	item := heap.Pop(&extPQ.pq).(*Item)

	req := UniqueCommand {
		senderId: item.entry.SenderId,
		term: item.entry.Term,
		index: item.entry.Index,
		time: item.entry.Timestamp,
	}

	delete(extPQ.itemLoc, req)
	return item
}

func (extPQ ExtendedPriorityQueue) peek() *Item {
	return extPQ.pq[0]
}

func (extPQ ExtendedPriorityQueue) clearLeaderEntries(){
	for len(extPQ.pq) > 0 && extPQ.peek().entry.Term != -1 {
		extPQ.pop()
	}
}

func (extPQ ExtendedPriorityQueue) extractList() []randomizedpaxosproto.Entry {
	var list []randomizedpaxosproto.Entry
	for i := 0; i < len(extPQ.pq); i++ {
		list = append(list, extPQ.pq[i].entry)
	}
	sort.Slice(list, func(i, j int) bool {
		return cmpEntry(list[i], list[j])
	})
	return list
}

func newExtendedPriorityQueue() ExtendedPriorityQueue {
	var extPQ ExtendedPriorityQueue
	extPQ.pq = make(PriorityQueue, 0)
	extPQ.itemLoc = make(map[UniqueCommand]*Item, 0)
	heap.Init(&extPQ.pq)
	return extPQ
}

// An Item is something we manage in a priority queue.
type Item struct {
	// senderId int32
	// term	 int32
	// index	 int32
	// time	 int64
	// request  state.Command
	entry randomizedpaxosproto.Entry

// 	value    string // The value of the item; arbitrary.
// 	priority int    // The priority of the item in the queue.
// 	// The index is needed by update and is maintained by the heap.Interface methods.
// 	index int // The index of the item in the heap.
	heapIndex int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func cmpEntry(a, b randomizedpaxosproto.Entry) bool {
	if a.Timestamp != b.Timestamp {
		return a.Timestamp < b.Timestamp
	}
	return a.SenderId < b.SenderId
	// if a.Term != b.Term {
	// 	return a.Term > b.Term
	// }
	// if a.Index != b.Index {
	// 	return a.Index < b.Index
	// }
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
func (pq *PriorityQueue) update(item *Item, entry randomizedpaxosproto.Entry, request state.Command) {
	item.entry = entry
	heap.Fix(pq, item.heapIndex)
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
