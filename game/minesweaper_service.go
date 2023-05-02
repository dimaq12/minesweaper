package game

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/dimaq12/minesweaper/models"
)

type ShowTask struct {
	Row int
	Col int
}

func NewShowTask(row, col int) *ShowTask {
	return &ShowTask{Row: row, Col: col}
}

type GameService interface {
	InitGame(bSize int, mineQ int)
	EndGame()
	showCell(row, col int)
	flagCell(row, col int)
	isGameOver() (bool, bool)
	handleInput(app *tview.Application)
}

type MinesweeperService struct {
	game            *models.Minesweeper
	logger          io.Writer
	renderer        *Renderer
	app             *tview.Application
	mineQuantity    int
	cancelFunc      context.CancelFunc
	showTasks       chan *ShowTask
	rerenderTasks   chan struct{}
	checkGameStatus chan struct{}
	revealAllBoard  chan struct{}
}

func NewMinesweeperService(game *models.Minesweeper) *MinesweeperService {
	renderer := NewRenderer()
	return &MinesweeperService{
		game:     game,
		renderer: renderer,
	}
}

func (s *MinesweeperService) InitGame(bSize int, mineQ int) {
	s.game = models.NewMinesweeper(bSize)
	s.game.PlaceMinesRandomly(mineQ)
	s.mineQuantity = mineQ
	s.renderer.DrawBoard(s.game)
	s.app = tview.NewApplication()
	s.app.SetRoot(s.renderer.boardTable, true)
	s.showTasks = make(chan *ShowTask)
	s.rerenderTasks = make(chan struct{})
	s.checkGameStatus = make(chan struct{})
	s.revealAllBoard = make(chan struct{})
	ctx, cancel := context.WithCancel(context.TODO())
	s.cancelFunc = cancel
	go s.run(ctx)

	s.handleInput()

	if err := s.app.Run(); err != nil {
		panic(err)
	}
}

func (s *MinesweeperService) EndGame() {
	s.app.Stop()
	s.cancelFunc()
	os.Exit(0)
}

// ifCellValid takes a cell's row and col coordinates as input and returns
// a boolean value indicating whether the given cell coordinates are within
// the game board's borders. This function is used to ensure that cell
// operations are only performed on valid cells within the game board.
func (s *MinesweeperService) ifCellValid(row, col int) bool {
	// Check if the given row and col are within the borders of the game board:
	// The row must be greater than or equal to 0 and less than the total number of rows
	// The col must be greater than or equal to 0 and less than the total number of columns
	return row >= 0 && row < s.game.Rows && col >= 0 && col < s.game.Cols
}

// countNearbyMines  takes a cell's row and col coordinates as input and returns
// the number of mines in the nearby cells. This function is used to calculate
// the number of mines around a cell and is called when a cell is shown.
func (s *MinesweeperService) countNearbyMines(row, col int) int {
	// Initialize the nearbyMines counter to 0
	nearbyMines := 0

	// Loop through the nearby cells by using deltaRow and deltaCol (delta)
	// deltaRow ranges from -1 to 1, representing the row above, the same row, and the row below
	for deltaRow := -1; deltaRow <= 1; deltaRow++ {
		// deltaCol ranges from -1 to 1, representing the column to the left, the same column, and the column to the right
		for deltaCol := -1; deltaCol <= 1; deltaCol++ {
			// If both deltaRow and deltaCol are 0, it means we are looking at the current cell, so skip this iteration
			if deltaRow == 0 && deltaCol == 0 {
				continue
			}

			// Calculate the nearby cell's row and col coordinates by adding deltaRow and deltaCol to the current row and col
			newRow, newCol := row+deltaRow, col+deltaCol

			// Check if the nearby cell's row and col are over the game board borders and if the cell contains a mine
			if s.ifCellValid(newRow, newCol) && s.game.Board[newRow][newCol].IsMine {
				// If the nearby cell contains a mine, increment the nearbyMines counter by 1
				nearbyMines++
			}
		}
	}

	// Return the total number of mines found in the nearby cells
	return nearbyMines
}

// showCell takes a cell's row and col coordinates as input and show
// the cell, updating its IsShown state and the number of nearby mines.
// If the shown cell has zero nearby mines, it recursively show
// all neighboring cells that are not already shown.
func (s *MinesweeperService) showCell(row, col int, recursive bool) {
	// Check if the given row and col are within the borders of the game board,
	// and if the cell is already shown. If either of these conditions is true,
	// the function returns immediately without revealing the cell.
	if !s.ifCellValid(row, col) || s.game.Board[row][col].IsShown {
		return
	}
	s.game.Mu.Lock()

	// Set the cell's IsShown property to true, indicating that it has been shown.
	if s.game.Board[row][col].IsFlagged == !true {
		s.game.Board[row][col].IsShown = true
	}

	// Update the cell's nearbyMines property with the count of nearby mines.
	s.game.Board[row][col].NearbyMines = s.countNearbyMines(row, col)
	s.game.Mu.Unlock()

	// If the shown cell has no nearby mines (i.e., nearbyMines is 0),
	// recursively reveal all neighboring cells.
	if s.game.Board[row][col].NearbyMines == 0 {
		// Loop through all neighboring cells using relative row (deltaRow) and column (deltaCol) offsets.
		for deltaRow := -1; deltaRow <= 1; deltaRow++ {
			for deltaCol := -1; deltaCol <= 1; deltaCol++ {
				// Skip the current cell 0, 0
				if deltaRow == 0 && deltaCol == 0 {
					continue
				}
				// Recursively call showCell for the neighboring cell.
				s.showCell(row+deltaRow, col+deltaCol, true)
			}
		}
	}
	s.rerenderTasks <- struct{}{}
	s.checkGameStatus <- struct{}{}
}

// Show all the cells on the board
func (s *MinesweeperService) revealAll() {
	s.game.Mu.Lock()
	for row := 0; row < s.game.Rows; row++ {
		for col := 0; col < s.game.Cols; col++ {
			s.game.Board[row][col].IsShown = true
		}
	}
	s.game.Mu.Unlock()
	s.rerenderTasks <- struct{}{}
}

func (s *MinesweeperService) isWinOrGameOver() (bool, bool) {
	shownNonMineCells := 0
	allCells := s.game.Rows * s.game.Cols
	s.game.Mu.Lock()
	for row := 0; row < s.game.Rows; row++ {
		for col := 0; col < s.game.Cols; col++ {
			cell := s.game.Board[row][col]
			if cell.IsShown {
				if cell.IsMine {
					// If a shown cell is a mine, the player has lost.
					s.game.Mu.Unlock()
					return true, false
				} else {
					shownNonMineCells++
				}
			}
		}
	}
	s.game.Mu.Unlock()

	// If all non-mine cells are shown, the player has won.
	if allCells-shownNonMineCells == s.mineQuantity {
		return true, true
	}

	// The game is still ongoing.
	return false, false
}

// Flag Cell
func (s *MinesweeperService) flagCell(row, col int) {
	if s.ifCellValid(row, col) {
		s.game.Board[row][col].IsFlagged = !s.game.Board[row][col].IsFlagged
	}
}

// Handle input
func (s *MinesweeperService) handleInput() {
	s.renderer.boardTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Get coordinate of input
		row, col := s.renderer.boardTable.GetSelection()

		switch event.Key() {
		// If enter was pressed
		case tcell.KeyEnter:
			s.showTasks <- NewShowTask(row, col) // Send a Show task

		// If F or Q was pressed
		case tcell.KeyRune:
			switch event.Rune() {
			case 'f', 'F':
				s.flagCell(row, col)
				s.rerenderTasks <- struct{}{}
			case 'q', 'Q':
				s.EndGame()
			}
		}
		return event
	})
}

// Run all listeners
func (s *MinesweeperService) run(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-s.showTasks:
				s.showCell(task.Row, task.Col, true)
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.rerenderTasks:
				s.app.QueueUpdateDraw(func() {
					s.renderer.DrawBoard(s.game)
				})
			}
		}

	}(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.revealAllBoard:
				s.revealAll()
			}
		}

	}(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.checkGameStatus:
				gameOver, gameWon := s.isWinOrGameOver()

				if gameOver {
					if gameWon {
						s.revealAllBoard <- struct{}{}
						time.Sleep(5 * time.Second)
						s.app.Stop()
						fmt.Println("Congratulations! You won the game!")
					} else {
						s.revealAllBoard <- struct{}{}
						time.Sleep(5 * time.Second)
						s.app.Stop()
						fmt.Println("Game Over! You hit a mine.")
					}
					os.Exit(0)
				}
			}
		}
	}(ctx)
}
