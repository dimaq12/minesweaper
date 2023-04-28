package game

import (
	"fmt"

	"github.com/dimaq12/minesweaper/models"
	"github.com/rivo/tview"
)

type Renderer struct {
	boardTable *tview.Table
}

func NewRenderer() *Renderer {
	return &Renderer{
		boardTable: tview.NewTable(),
	}
}

func (r *Renderer) DrawBoard(game *models.Minesweeper) {
	for row := 0; row < game.Rows; row++ {
		for col := 0; col < game.Cols; col++ {
			r.RenderCell(game, row, col)
		}
	}

	r.boardTable.SetSelectable(true, true)
	r.boardTable.SetFixed(game.Rows, game.Cols)
}

func (r *Renderer) RenderCell(game *models.Minesweeper, row, col int) {
	cell := game.Board[row][col]

	cellText := "."
	if cell.IsShown {
		if cell.IsMine {
			cellText = "M"
		} else {
			cellText = fmt.Sprintf("%d", cell.NearbyMines)
		}
	} else if cell.IsFlagged {
		cellText = "F"
	}

	r.boardTable.SetCell(row, col, tview.NewTableCell(cellText).SetAlign(tview.AlignCenter))
}
