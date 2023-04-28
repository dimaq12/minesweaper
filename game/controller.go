package game

import (
	"fmt"
)

type GameController struct {
	service *MinesweeperService
}

func NewGameController(service *MinesweeperService) *GameController {
	return &GameController{service: service}
}

func (c *GameController) StartGame(boardSize, mineQuantity int) {
	c.service.InitGame(boardSize, mineQuantity)
}

func (c *GameController) TerminateGame() {
	fmt.Println("Terminating the game...")
}
