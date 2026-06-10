package game

type Board struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func NewBoard() *Board {
	return &Board{
		Width:  BoardWidth,
		Height: BoardHeight,
	}
}

type Score struct {
	Player1 int `json:"player1"`
	Player2 int `json:"player2"`
}

func NewScore() *Score {
	return &Score{
		Player1: 0,
		Player2: 0,
	}
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func NewPosition(x, y float64) *Position {
	return &Position{
		X: x,
		Y: y,
	}
}

type Vector struct {
	VX float64 `json:"vx"`
	VY float64 `json:"vy"`
}

func NewVector(vx, vy float64) *Vector {
	return &Vector{
		VX: vx,
		VY: vy,
	}
}

type Paddle struct {
	PlayerID PlayerID  `json:"playerID"`
	Pos      *Position `json:"position"`
	Velocity *Vector   `json:"velocity"`
	Radius   float64   `json:"radius"`
}

func NewPaddle(playerID PlayerID, x, y float64) *Paddle {
	return &Paddle{
		PlayerID: playerID,
		Pos:      NewPosition(x, y),
		Velocity: NewVector(0, 0),
		Radius:   PaddleRadius,
	}
}

type Puck struct {
	Pos      Position `json:"position"`
	Velocity Vector   `json:"velocity"`
	Radius   float64  `json:"radius"`
}

func NewPuck(x, y float64) *Puck {
	return &Puck{
		Pos:      *NewPosition(x, y),
		Velocity: *NewVector(0, 0),
		Radius:   PuckRadius,
	}
}
