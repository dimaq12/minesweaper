package main

import (
	"flag"
	"fmt"

	"github.com/dimaq12/minesweaper/game"
	"github.com/dimaq12/minesweaper/models"
)

func main() {
	boardSize := 9
	mineQuantity := 10
	flag.Parse()

	if mineQuantity >= boardSize*boardSize {
		fmt.Println("The number of mines must be less than the total number of cells.")
		return
	}

	minesweeperGame := models.NewMinesweeper(boardSize, boardSize)
	minesweeperService := game.NewMinesweeperService(minesweeperGame)

	minesweeperService.InitGame(boardSize, mineQuantity)
}
