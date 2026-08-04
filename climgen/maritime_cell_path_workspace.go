package climgen

import "math"

const maxIntValue = int(^uint(0) >> 1)

type maritimeCellPathWorkspace struct {
	dist       []float64
	prev       []int
	mark       []int
	generation int
}

func newMaritimeCellPathWorkspace(cellCount int) *maritimeCellPathWorkspace {
	w := &maritimeCellPathWorkspace{}
	w.ensure(cellCount)
	return w
}

func (w *maritimeCellPathWorkspace) begin(cellCount int) {
	w.ensure(cellCount)
	if w.generation == maxIntValue {
		for i := range w.mark {
			w.mark[i] = 0
		}
		w.generation = 0
	}
	w.generation++
}

func (w *maritimeCellPathWorkspace) ensure(cellCount int) {
	if w == nil || cellCount < 0 {
		return
	}
	if len(w.dist) == cellCount && len(w.prev) == cellCount && len(w.mark) == cellCount {
		return
	}
	w.dist = make([]float64, cellCount)
	w.prev = make([]int, cellCount)
	w.mark = make([]int, cellCount)
}

func (w *maritimeCellPathWorkspace) getDist(cell int) float64 {
	if w == nil || cell < 0 || cell >= len(w.dist) || w.mark[cell] != w.generation {
		return math.Inf(1)
	}
	return w.dist[cell]
}

func (w *maritimeCellPathWorkspace) setDist(cell int, cost float64, prev int) {
	if w == nil || cell < 0 || cell >= len(w.dist) {
		return
	}
	w.mark[cell] = w.generation
	w.dist[cell] = cost
	w.prev[cell] = prev
}

func (w *maritimeCellPathWorkspace) prevCell(cell int) int {
	if w == nil || cell < 0 || cell >= len(w.prev) || w.mark[cell] != w.generation {
		return -1
	}
	return w.prev[cell]
}
