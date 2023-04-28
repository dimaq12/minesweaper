package models

import (
	"math/rand"
	"time"
)

type Cell struct {
	IsMine      bool
	IsShown     bool
	IsFlagged   bool
	NearbyMines int
}

type Minesweeper struct {
	Board [][]Cell
	Rows  int
	Cols  int
}

func NewMinesweeper(rows int, cols int) *Minesweeper {
	board := make([][]Cell, rows)
	for i := range board {
		board[i] = make([]Cell, cols)
	}

	return &Minesweeper{
		Board: board,
		Rows:  rows,
		Cols:  cols,
	}
}

// PlaceMinesRandomly places N mines randomly on the game board.
func (ms *Minesweeper) PlaceMinesRandomly(N int) {
	// Step 1: Create a list containing the coordinates of all the cells on the board.
	// Create a slice of [2]int, where each element represents a cell's coordinates.
	coords := make([][2]int, ms.Rows*ms.Cols)
	// Iterate through each row of the game board.
	for row := 0; row < ms.Rows; row++ {
		// Iterate through each column of the game board.
		for col := 0; col < ms.Cols; col++ {
			// Store the current row and column in the 'coords' slice.
			coords[row*ms.Cols+col] = [2]int{row, col}
		}
	}

	// Step 2: Shuffle the list using the Fisher-Yates shuffle algorithm.
	// https://en.wikipedia.org/wiki/Fisher–Yates_shuffle
	// Seed the random number generator using the current Unix timestamp.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Iterate through the 'coords' slice in reverse.
	for i := len(coords) - 1; i > 0; i-- {
		// Generate a random index 'j' within the range [0, i].
		j := r.Intn(i + 1)
		// Swap the elements at indices i and j in the 'coords' slice.
		coords[i], coords[j] = coords[j], coords[i]
	}

	// Step 3: Place mines in the first N cells from the shuffled list.
	// Iterate through the first N elements of the shuffled 'coords' slice.
	for i := 0; i < N && i < len(coords); i++ {
		// Extract the row and column from the current coordinate.
		row, col := coords[i][0], coords[i][1]
		// Set the 'IsMine' field of the cell at the current coordinate to 'true'.
		ms.Board[row][col].IsMine = true
	}
}
