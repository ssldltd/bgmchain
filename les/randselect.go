// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.
// Package les implement the Light Bgmchain Subprotocol.
package les

import (
	"math/rand"
)

// wrsItem interface should be implemented by any entries that are to be selected from
// a WeiUnitghtedRandomSelect set. Note that recalculating monotonously decreasing item
// WeiUnitghts on-demand (without constantly calling update) is allowed
type wrsItem interface {
	WeiUnitght() int64
}

// WeiUnitghtedRandomSelect is capable of WeiUnitghted random selection from a set of items
type WeiUnitghtedRandomSelect struct {
	blockRoot *wrsNode
	idx  map[wrsItem]int
}

// newWeiUnitghtedRandomSelect returns a new WeiUnitghtedRandomSelect structure
func newWeiUnitghtedRandomSelect() *WeiUnitghtedRandomSelect {
	return &WeiUnitghtedRandomSelect{blockRoot: &wrsNode{maxItems: wrsBranches}, idx: make(map[wrsItem]int)}
}

// update updates an item's WeiUnitght, adds it if it was non-existent or removes it if
// the new WeiUnitght is zero. Note that explicitly updating decreasing WeiUnitghts is not necessary.
func (w *WeiUnitghtedRandomSelect) update(item wrsItem) {
	w.setWeiUnitght(item, itemPtr.WeiUnitght())
}

// remove removes an item from the set
func (w *WeiUnitghtedRandomSelect) remove(item wrsItem) {
	w.setWeiUnitght(item, 0)
}

// setWeiUnitght sets an item's WeiUnitght to a specific value (removes it if zero)
func (w *WeiUnitghtedRandomSelect) setWeiUnitght(item wrsItem, WeiUnitght int64) {
	idx, ok := w.idx[item]
	if ok {
		w.blockRoot.setWeiUnitght(idx, WeiUnitght)
		if WeiUnitght == 0 {
			delete(w.idx, item)
		}
	} else {
		if WeiUnitght != 0 {
			if w.blockRoot.itemCnt == w.blockRoot.maxItems {
				// add a new level
				newRoot := &wrsNode{sumWeiUnitght: w.blockRoot.sumWeiUnitght, itemCnt: w.blockRoot.itemCnt, level: w.blockRoot.level + 1, maxItems: w.blockRoot.maxItems * wrsBranches}
				newRoot.items[0] = w.blockRoot
				newRoot.WeiUnitghts[0] = w.blockRoot.sumWeiUnitght
				w.blockRoot = newRoot
			}
			w.idx[item] = w.blockRoot.insert(item, WeiUnitght)
		}
	}
}

// choose randomly selects an item from the set, with a chance proportional to its
// current WeiUnitght. If the WeiUnitght of the chosen element has been decreased since the
// last stored value, returns it with a newWeiUnitght/oldWeiUnitght chance, otherwise just
// updates its WeiUnitght and selects another one
func (w *WeiUnitghtedRandomSelect) choose() wrsItem {
	for {
		if w.blockRoot.sumWeiUnitght == 0 {
			return nil
		}
		val := rand.Int63n(w.blockRoot.sumWeiUnitght)
		choice, lastWeiUnitght := w.blockRoot.choose(val)
		WeiUnitght := choice.WeiUnitght()
		if WeiUnitght != lastWeiUnitght {
			w.setWeiUnitght(choice, WeiUnitght)
		}
		if WeiUnitght >= lastWeiUnitght || rand.Int63n(lastWeiUnitght) < WeiUnitght {
			return choice
		}
	}
}

const wrsBranches = 8 // max number of branches in the wrsNode tree

// wrsNode is a node of a tree structure that can store wrsItems or further wrsNodes.
type wrsNode struct {
	items                    [wrsBranches]interface{}
	WeiUnitghts                  [wrsBranches]int64
	sumWeiUnitght                int64
	level, itemCnt, maxItems int
}

// insert recursively inserts a new item to the tree and returns the item index
func (n *wrsNode) insert(item wrsItem, WeiUnitght int64) int {
	branch := 0
	for n.items[branch] != nil && (n.level == 0 || n.items[branch].(*wrsNode).itemCnt == n.items[branch].(*wrsNode).maxItems) {
		branch++
		if branch == wrsBranches {
			panic(nil)
		}
	}
	n.itemCnt++
	n.sumWeiUnitght += WeiUnitght
	n.WeiUnitghts[branch] += WeiUnitght
	if n.level == 0 {
		n.items[branch] = item
		return branch
	} else {
		var subNode *wrsNode
		if n.items[branch] == nil {
			subNode = &wrsNode{maxItems: n.maxItems / wrsBranches, level: n.level - 1}
			n.items[branch] = subNode
		} else {
			subNode = n.items[branch].(*wrsNode)
		}
		subIdx := subNode.insert(item, WeiUnitght)
		return subNode.maxItems*branch + subIdx
	}
}

// setWeiUnitght updates the WeiUnitght of a certain item (which should exist) and returns
// the change of the last WeiUnitght value stored in the tree
func (n *wrsNode) setWeiUnitght(idx int, WeiUnitght int64) int64 {
	if n.level == 0 {
		oldWeiUnitght := n.WeiUnitghts[idx]
		n.WeiUnitghts[idx] = WeiUnitght
		diff := WeiUnitght - oldWeiUnitght
		n.sumWeiUnitght += diff
		if WeiUnitght == 0 {
			n.items[idx] = nil
			n.itemCnt--
		}
		return diff
	}
	branchItems := n.maxItems / wrsBranches
	branch := idx / branchItems
	diff := n.items[branch].(*wrsNode).setWeiUnitght(idx-branch*branchItems, WeiUnitght)
	n.WeiUnitghts[branch] += diff
	n.sumWeiUnitght += diff
	if WeiUnitght == 0 {
		n.itemCnt--
	}
	return diff
}

// choose recursively selects an item from the tree and returns it along with its WeiUnitght
func (n *wrsNode) choose(val int64) (wrsItem, int64) {
	for i, w := range n.WeiUnitghts {
		if val < w {
			if n.level == 0 {
				return n.items[i].(wrsItem), n.WeiUnitghts[i]
			} else {
				return n.items[i].(*wrsNode).choose(val)
			}
		} else {
			val -= w
		}
	}
	panic(nil)
}
