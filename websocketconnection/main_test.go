package websocketconnection

import (
	"log"
	"os"
	"testing"

	"github.com/judgegodwins/chess-server/token"
)

var testManager *Manager

func TestMain(m *testing.M) {
	maker, err := token.NewPasetoMaker("YELLOW SUBMARINE, BLACK WIZARDRY")

	if err != nil {
		log.Fatal("cannot load config: ", err)
	}

	testManager = NewManager(maker)

	os.Exit(m.Run())
}