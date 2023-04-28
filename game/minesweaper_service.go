package game

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/dimaq12/minesweaper/models"
)

type TaskType int

const (
	RenderTaskType TaskType = iota
	ShowTaskType
	RevealAllTaskType
)

type Task struct {
	Type TaskType
	Row  int
	Col  int
}

func NewTask(taskType TaskType, row, col int) *Task {
	return &Task{Type: taskType, Row: row, Col: col}
}

type GameService interface {
	InitGame(bSize int, mineQ int)
	showCell(row, col int)
	flagCell(row, col int)
	isGameOver() (bool, bool)
	handleInput(app *tview.Application)
}

type MinesweeperService struct {
	game         *models.Minesweeper
	renderer     *Renderer
	app          *tview.Application
	mineQuantity int
	tasks        chan *Task
}

func NewMinesweeperService(game *models.Minesweeper) *MinesweeperService {
	renderer := NewRenderer()
	return &MinesweeperService{
		game:     game,
		renderer: renderer,
	}
}

func (s *MinesweeperService) InitGame(bSize int, mineQ int) {
	s.game = models.NewMinesweeper(bSize, bSize)
	s.game.PlaceMinesRandomly(mineQ)
	s.mineQuantity = mineQ
	s.renderer.DrawBoard(s.game)
	s.app = tview.NewApplication()
	s.app.SetRoot(s.renderer.boardTable, true)
	s.tasks = make(chan *Task)
	go s.taskPipeline()

	s.handleInput()

	if err := s.app.Run(); err != nil {
		panic(err)
	}
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
func (s *MinesweeperService) showCell(row, col int) {
	// Check if the given row and col are within the borders of the game board,
	// and if the cell is already shown. If either of these conditions is true,
	// the function returns immediately without revealing the cell.
	if !s.ifCellValid(row, col) || s.game.Board[row][col].IsShown {
		return
	}

	// Set the cell's IsShown property to true, indicating that it has been shown.
	s.game.Board[row][col].IsShown = true
	// Update the cell's nearbyMines property with the count of nearby mines.
	s.game.Board[row][col].NearbyMines = s.countNearbyMines(row, col)

	// Send a RenderTaskType task to the task pipeline to update the UI for the shown cell.
	// To-do fix the issue with deadlock
	s.tasks <- NewTask(RenderTaskType, row, col)

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
				go s.showCell(row+deltaRow, col+deltaCol)
			}
		}
	}
}

// Show all the cells on the board
func (s *MinesweeperService) revealAll() {
	for row := 0; row < s.game.Rows; row++ {
		for col := 0; col < s.game.Cols; col++ {
			s.game.Board[row][col].IsShown = true
			s.tasks <- NewTask(RenderTaskType, row, col) // Send a Render task
		}
	}
}

func (s *MinesweeperService) isGameOver() (bool, bool) {
	unShownNonMineCells := 0
	for row := 0; row < s.game.Rows; row++ {
		for col := 0; col < s.game.Cols; col++ {
			cell := s.game.Board[row][col]
			if cell.IsShown {
				if cell.IsMine {
					// If a shown cell is a mine, the player has lost.
					return true, false
				} else {
					unShownNonMineCells++
				}
			}
		}
	}

	// If all non-mine cells are shown, the player has won.
	if unShownNonMineCells == s.mineQuantity {
		return true, true
	}

	// The game is still ongoing.
	return false, false
}

func (s *MinesweeperService) flagCell(row, col int) {
	if s.ifCellValid(row, col) {
		s.game.Board[row][col].IsFlagged = !s.game.Board[row][col].IsFlagged
	}
}

func (s *MinesweeperService) taskPipeline() {
	for task := range s.tasks {
		switch task.Type {
		case ShowTaskType:
			go s.showCell(task.Row, task.Col)
			fallthrough
		case RenderTaskType:
			go s.renderer.RenderCell(s.game, task.Row, task.Col)
		case RevealAllTaskType:
			s.revealAll()
		}
	}
}

// To-do fix the issue with rerender in the case when game is lost
func (s *MinesweeperService) handleInput() {
	s.renderer.boardTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, col := s.renderer.boardTable.GetSelection()

		switch event.Key() {
		case tcell.KeyEnter:
			s.tasks <- NewTask(ShowTaskType, row, col) // Send a Reveal task
			gameOver, gameWon := s.isGameOver()
			if gameOver {
				if gameWon {
					time.Sleep(1 * time.Second)
					s.app.Stop()
					fmt.Println("Congratulations! You won the game!")
				} else {
					s.tasks <- NewTask(RevealAllTaskType, row, col)
					time.Sleep(1 * time.Second)
					s.app.Stop()
					fmt.Println("Game Over! You hit a mine.")
				}
				os.Exit(0)
			}

		case tcell.KeyRune:
			switch event.Rune() {
			case 'f', 'F':
				s.flagCell(row, col)
			}
		}
		s.tasks <- NewTask(RenderTaskType, row, col) // Send a Render task

		return event
	})
}
