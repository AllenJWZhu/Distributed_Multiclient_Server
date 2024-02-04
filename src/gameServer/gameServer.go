package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	RunningProtocol      string = "tcp"
	ServerAddress        string = "localhost:9999"
	StorageDirectoryName string = "serverStorage/"
	MIN_PLAYERS          int    = 4
	MAX_PLAYERS          int    = 8
)

var RootDir, _ = os.Getwd()

// GameServer holds the structure of our word count game server
// implementation.
type GameServer struct {
	addr    string
	players map[string]*Player
	games   map[string]*Game

	chanName   chan string  // player sends name to server ...
	chanPlayer chan *Player // ... and receives a Player object

	chanGameReq chan struct {
		gameID  string
		name    string
		newGame bool
	} // player sends a request for a game (existing or new) ...
	chanGameResp chan chan map[string]string // .. and receives its mailbox

	chanPlayerReq  chan string                 // game sends a player name to server ...
	chanPlayerResp chan chan map[string]string // ... and receives its mailbox

	chanGameExit     chan string // gameID of a exited game
	chanGameExitResp chan bool
	chanPlayerExit   chan string // name of a exited player
	chanShutdown     chan bool   // shut down game server

	directory string // storage directory
	listener  *net.Listener
}

func (server *GameServer) Run() (err error) {
	listener, err := net.Listen(RunningProtocol, ServerAddress)
	if err != nil {
		fmt.Println(err)
		return
	}
	server.listener = &listener
	// launch a routine to accept TCP connections and dispatch them to clientRoutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}
			go clientRoutine(conn, server)
		}
	}()

	// server routine provides services to games and players outside a game
loop:
	for {
		select {
		case name := <-server.chanName:
			player, ok := server.players[name]
			if !ok {
				// a new player
				player = server.newPlayer(name)
			}
			server.chanPlayer <- player

		case req := <-server.chanGameReq:
			game, ok := server.games[req.gameID]
			if req.newGame && !ok {
				// create a new game
				game = server.newGame(req.gameID, req.name)
				server.chanGameResp <- game.mailbox
			} else if !req.newGame && ok {
				// join an existing game
				server.chanGameResp <- game.mailbox
			} else {
				// create an existing game, or join a non-exist game
				server.chanGameResp <- nil
			}

		case req := <-server.chanPlayerReq:
			player := server.players[req]
			server.chanPlayerResp <- player.mailbox

		case gameID := <-server.chanGameExit:
			delete(server.games, gameID)
			server.chanGameExitResp <- true

		case name := <-server.chanPlayerExit:
			delete(server.players, name)

		case <-server.chanShutdown:
			break loop
		}
	}
	(*server.listener).Close()
	for _, game := range server.games {
		game.exit <- true
		<-game.exit
	}
	server.chanShutdown <- true
	return nil
}

// Close shuts down the game server from another routine
func (server *GameServer) Close() (err error) {
	server.chanShutdown <- true
	<-server.chanShutdown
	return nil
}

// called by GameServer to initiate a new player
func (server *GameServer) newPlayer(name string) *Player {
	player := Player{
		name:    name,
		gameIDs: make(map[string]chan map[string]string),
		mailbox: make(chan map[string]string),
		server:  server}
	server.players[name] = &player
	return &player
}

// called by GameServer to initiate a new game
func (server *GameServer) newGame(gameID string, leader string) *Game {
	game := Game{
		gameID:       gameID,
		state:        WAITING,
		leader:       leader,
		names:        make(map[string]chan map[string]string),
		namesDisconn: make(map[string]chan map[string]string),
		namesBye:     make(map[string]chan map[string]string),
		namesOrd:     make(map[string]int),
		usedWords:    make(map[string]bool),
		wordDict:     make(map[string]int),
		mailbox:      make(chan map[string]string),
		exit:         make(chan bool),
		guessResults: make(map[string]int),
		directory:    server.directory + gameID + "/",
		server:       server}
	player := server.players[leader]
	game.names[leader] = player.mailbox
	game.namesOrd[leader] = 0
	server.games[gameID] = &game
	os.Mkdir(game.directory, os.ModePerm)
	go game.routine()
	return &game
}

// Server defines the minimum contract our
// Game server implementations must satisfy.
type Server interface {
	Run() error
	Close() error
}

// NewServer creates a new Server using given protocol
// and addr.
func NewServer(protocol, addr string, directory string) (Server, error) {
	if strings.ToLower(protocol) != RunningProtocol {
		return nil, errors.New("invalid protocol given")
	}

	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return nil, errors.New("unable to create directories for the given path")
	}
	return &GameServer{
		addr:       addr,
		players:    make(map[string]*Player),
		games:      make(map[string]*Game),
		chanName:   make(chan string),
		chanPlayer: make(chan *Player),
		chanGameReq: make(chan struct {
			gameID  string
			name    string
			newGame bool
		}),
		chanGameResp:     make(chan chan map[string]string),
		chanPlayerReq:    make(chan string),
		chanPlayerResp:   make(chan chan map[string]string),
		chanGameExit:     make(chan string),
		chanGameExitResp: make(chan bool),
		chanPlayerExit:   make(chan string),
		chanShutdown:     make(chan bool),
		directory:        directory,
	}, nil
}

func main() {
	addrPtr := flag.String("port", ServerAddress, "Listening address for the game server")
	flag.Parse()

	// Start the new server
	gameServer, err := NewServer(RunningProtocol, *addrPtr, RootDir+"/"+StorageDirectoryName)
	if err != nil {
		log.Println("error starting the game server")
		return
	}
	// Run the servers
	gameServer.Run()
}
