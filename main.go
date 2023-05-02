package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dimaq12/minesweaper/game"
	"github.com/dimaq12/minesweaper/models"
)

func boardDimensions(level int) (boardSize, mineQuantity int) {
	switch level {
	case 1:
		return 10, 10 // 10x10 board with 10 mines
	case 2:
		return 15, 40 // 15x15 board with 40 mines
	case 3:
		return 20, 80 // 20x20 board with 80 mines
	case 4:
		return 25, 125 // 25x25 board with 125 mines
	case 5:
		return 30, 180 // 30x30 board with 180 mines
	default:
		return 10, 10 // Default to 10x10 board with 10 mines for invalid level input
	}
}

func main() {
	var input string
	var level int
	var err error

	for {
		fmt.Print("Enter the level (1-5) or 'q' to quit: ")
		_, err = fmt.Scan(&input)

		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		if strings.ToLower(input) == "q" {
			fmt.Println("Quitting...")
			return
		}

		level, err = strconv.Atoi(input)
		if err == nil && level >= 1 && level <= 5 {
			break
		}

		fmt.Println("Invalid input. Please enter a level between 1 and 5 or 'q' to quit.")
	}

	fmt.Println("Level:", level)

	bSize, mineQ := boardDimensions(level)

	minesweeperGame := models.NewMinesweeper(bSize)
	minesweeperService := game.NewMinesweeperService(minesweeperGame)

	minesweeperService.InitGame(bSize, mineQ)
}
