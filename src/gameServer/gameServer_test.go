package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	MIN_SLEEP int = 10
	MAX_SLEEP int = 50
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randSleep() {
	rand.Seed(time.Now().UnixNano())
	randNumber := rand.Intn(MAX_SLEEP-MIN_SLEEP+1) + MIN_SLEEP
	time.Sleep(time.Duration(randNumber) * time.Millisecond)
}

type TestServer struct {
	protocol   string
	addr       string
	gameServer Server
}

func NewTestServer(t *testing.T) *TestServer {
	rand.Seed(time.Now().UnixNano())
	// Start the new server
	os.RemoveAll(RootDir + StorageDirectoryName)
	gameServer, err := NewServer(RunningProtocol, ServerAddress, RootDir+StorageDirectoryName)
	if err != nil || gameServer == nil {
		t.Fatalf("Error in server creation: %v", err.Error())
	}
	// Run the servers in goroutines to stop blocking
	go func() {
		gameServer.Run()
	}()
	randSleep()
	return &TestServer{RunningProtocol, ServerAddress, gameServer}
}

func (tg *TestServer) CleanUp(t *testing.T) {
	err := os.RemoveAll(RootDir + StorageDirectoryName)
	if err != nil {
		t.Fatalf("Error in storage deletion: %v", err.Error())
	}
	tg.gameServer.Close()
}

func (ts *TestServer) Connect(t *testing.T) net.Conn {
	conn, err := net.Dial(ts.protocol, ts.addr)
	if err != nil || conn == nil {
		t.Fatalf("Error in connection: %v", err.Error())
	}
	return conn
}

type TestPlayer struct {
	name string
	conn net.Conn
}

func NewPlayer(t *testing.T, ts *TestServer, i int) *TestPlayer {
	player := &TestPlayer{}
	player.name = fmt.Sprintf("Player%d", i)
	player.conn = ts.Connect(t)

	return player
}

func NewEmptyPlayer(t *testing.T, ts *TestServer) *TestPlayer {
	player := &TestPlayer{}
	player.name = ""
	player.conn = ts.Connect(t)

	return player
}

func (tp *TestPlayer) Close() {
	tp.conn.Close()
}

func (tp *TestPlayer) SendInvalid(t *testing.T) {
	payload := "INVALID\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendHello(t *testing.T) {
	payload := "HELLO " + tp.name + "\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendNewGame(t *testing.T, tag string) {
	payload := "NEW_GAME " + tag + "\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendJoinGame(t *testing.T, tag string) {
	payload := "JOIN_GAME " + tag + "\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendStartGame(t *testing.T, tag string) {
	payload := "START_GAME " + tag + "\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendFileUpload(t *testing.T, tag, fileName string, fileSize int64) {
	pwd, _ := os.Getwd()
	contents, err := ioutil.ReadFile(strings.TrimSpace(pwd + "/" + fileName))
	if err != nil {
		t.Fatalf("Error in file read: %v", err.Error())
	}
	payload := fmt.Sprintf("FILE_UPLOAD %s %s %d %s\n", tag, fileName, fileSize, (string)(contents))
	_, err = tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendRandomWord(t *testing.T, tag, randomWord string) {
	payload := fmt.Sprintf("RANDOM_WORD %s %s\n", tag, randomWord)
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendGuessCount(t *testing.T, tag string, guess int64) {
	payload := fmt.Sprintf("WORD_COUNT %s %d\n", tag, guess)
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendRestart(t *testing.T, tag string) {
	payload := fmt.Sprintf("RESTART %s\n", tag)
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendClose(t *testing.T, tag string) {
	payload := fmt.Sprintf("CLOSE %s\n", tag)
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) SendGoodbye(t *testing.T) {
	payload := "GOODBYE\n"
	_, err := tp.conn.Write([]byte(payload))
	if err != nil {
		t.Fatalf("Error in write: %v", err.Error())
	}
}

func (tp *TestPlayer) ReadResponse(t *testing.T) string {
	out := make([]byte, 1024)
	tp.conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	n, err := tp.conn.Read(out)
	if err != nil {
		t.Fatalf("Error in read: %v", err.Error())
	}
	return strings.TrimSpace(string(out[:n]))
}

type TestGame struct {
	playerCount int
	players     []*TestPlayer
	server      *TestServer
	tag         string
	fileName    string
	fileSize    int64
	picker      *TestPlayer
}

func NewTestGame(t *testing.T, playerCount int) *TestGame {
	testServer := NewTestServer(t)
	testGame := &TestGame{
		players:     make([]*TestPlayer, 0),
		server:      testServer,
		playerCount: playerCount,
		fileName:    "test.txt",
		fileSize:    0,
	}
	return testGame
}

func (tg *TestGame) GetFileSize(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error in getwd: %v", err.Error())
	}
	file, err := os.Open(strings.TrimSpace(pwd + "/" + tg.fileName))
	if err != nil {
		t.Fatalf("Error in file open: %v", err.Error())
	}
	f, err := file.Stat()
	if err != nil {
		t.Fatalf("Error in file stat: %v", err.Error())
	}
	tg.fileSize = f.Size()
}

func (tg *TestGame) GameSetup(t *testing.T) {
	for i := 0; i < tg.playerCount; i++ {
		time.Sleep(10 * time.Millisecond)
		player := NewPlayer(t, tg.server, i)
		tg.players = append(tg.players, player)
		player.SendHello(t)
		resp := player.ReadResponse(t)
		expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Do you want to create a new game or join an existing game?", player.name)

		if resp != expectedResponse {
			t.Fatalf("Incorrect response in GameSetup HELLO")
		}
	}
}

// Player count should be atleast 2
func (tg *TestGame) NewGame(t *testing.T) {
	leader := tg.players[0]
	tg.tag = randSeq(6)
	leader.SendNewGame(t, tg.tag)
	resp := leader.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Game %s created! You are the leader of the game. Waiting for players to join.", tg.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response in NewGame")
	}

	leader.SendNewGame(t, tg.tag)
	badResp := leader.ReadResponse(t)
	expectedErrorResponse := fmt.Sprintf("Game %s already exists, please provide a new game tag.", tg.tag)
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect error response in NewGame")
	}
}

func (tg *TestGame) JoinGame(t *testing.T) {
	badPlayer := tg.players[1]
	badPlayer.SendJoinGame(t, "BAD")
	badResp := badPlayer.ReadResponse(t)
	expectedErrorResponse := "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect error response in JoinGame")
	}

	gameReady := false
	for j, player := range tg.players {
		if j != 0 {
			time.Sleep(10 * time.Millisecond)
			player.SendJoinGame(t, tg.tag)
			resp := player.ReadResponse(t)
			if j < MIN_PLAYERS-1 {
				expectedResponse := fmt.Sprintf("Joined Game %s. Current state is WAITING.", tg.tag)
				if resp != expectedResponse {
					t.Fatalf("Incorrect response in JoinGame WAITING")
				}
			} else if j == MAX_PLAYERS-1 {
				expectedResponse := fmt.Sprintf("Joined Game %s. Current state is FULL.", tg.tag)
				if resp != expectedResponse {
					t.Fatalf("Incorrect response in JoinGame FULL")
				}
				gameReady = true
			} else if j >= MAX_PLAYERS {
				expectedResponse := fmt.Sprintf("Game %s is full or already in progress. Connect back later.", tg.tag)
				if resp != expectedResponse {
					t.Fatalf("Incorrect response in JoinGame blocked")
				}
				gameReady = true
			} else {
				expectedResponse := fmt.Sprintf("Joined Game %s. Current state is READY.", tg.tag)
				if resp != expectedResponse {
					t.Fatalf("Incorrect response in JoinGame READY")
				}
				gameReady = true
			}
		}
	}

	if gameReady {
		leader := tg.players[0]
		resp := leader.ReadResponse(t)
		expectedResponse := fmt.Sprintf("Game %s is ready to start.", tg.tag)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response in JoinGame start")
		}
	}
}

func (tg *TestGame) StartGame(t *testing.T) {
	leader := tg.players[0]
	leader.SendStartGame(t, tg.tag)
	resp := leader.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Game %v is running. Please upload the file.", tg.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response in StartGame upload")
	}

	for j, player := range tg.players {
		if j != 0 {
			time.Sleep(10 * time.Millisecond)
			resp := player.ReadResponse(t)
			expectedResponse := fmt.Sprintf("Game %v has started. Waiting for %v to upload the file.", tg.tag, leader.name)
			if resp != expectedResponse {
				t.Fatalf("Incorrect response in StartGame waiting")
			}
		}
	}
}

func TestCheckpoint_Connection(t *testing.T) {
	// Simply check that the server is up and can
	// accept connections.
	playerCount := 8
	var wg sync.WaitGroup
	server := NewTestServer(t)
	for i := 0; i < playerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Need to introduce random sleep to avoid concurrent connection
			// which can cause connection refusal by the server
			randSleep()
			conn := server.Connect(t)
			defer conn.Close()
		}()
	}
	wg.Wait()
	server.CleanUp(t)
	// testGame.CleanUp(t)
}

func TestCheckpoint_Hello(t *testing.T) {
	testGame := NewTestGame(t, 1)
	testGame.GameSetup(t)

	player := NewEmptyPlayer(t, testGame.server)
	player.SendHello(t)
	resp := player.ReadResponse(t)
	expectedErrorResponse := "Invalid user name. Try again."
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to HELLO with invalid user name")
	}

	testGame.server.CleanUp(t)
}

func TestCheckpoint_NewGame(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)

	player := NewEmptyPlayer(t, testGame.server)
	player.SendNewGame(t, testGame.tag)
	resp := player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to NEW_GAME without HELLO")
	}

	testGame.server.CleanUp(t)
}

func TestCheckpoint_JoinGame(t *testing.T) {
	testGame := NewTestGame(t, 10)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	flakyClient := testGame.players[1]
	flakyClient.Close()
	flakyClient.conn = testGame.server.Connect(t)
	flakyClient.SendJoinGame(t, testGame.tag)
	resp := flakyClient.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to JOIN_GAME without HELLO")
	}
	testGame.server.CleanUp(t)
}

func TestCheckpoint_StartGame(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)

	leader := testGame.players[0]
	badPlayer := testGame.players[1]
	badPlayer.SendStartGame(t, "BAD")
	badResp := badPlayer.ReadResponse(t)
	expectedErrorResponse := "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect response to START_GAME with invalid tag")
	}

	badPlayer.SendStartGame(t, testGame.tag)
	badResp = badPlayer.ReadResponse(t)
	expectedErrorResponse = fmt.Sprintf("Only the leader can start the game. Please contact %v.", leader.name)
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect response to START_GAME by non-leader")
	}

	leader.SendStartGame(t, testGame.tag)
	expectedErrorResponse = fmt.Sprintf("Can't start the game %s, waiting for 3 more players.", testGame.tag)
	badResp = leader.ReadResponse(t)
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect response to START_GAME with too few players")
	}

	testGame.JoinGame(t)
	testGame.StartGame(t)
	leader.SendStartGame(t, testGame.tag)
	badResp = leader.ReadResponse(t)
	expectedErrorResponse = fmt.Sprintf("Game %s has already started! Please create a new game.", testGame.tag)
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect response to START_GAME when game already started")
	}

	testGame.server.CleanUp(t)
}

func TestFinal_FileUpload(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	leader := testGame.players[0]

	leader.Close()
	tobeLeader := testGame.players[1]
	resp := tobeLeader.ReadResponse(t)
	expectedResponse := fmt.Sprintf("You are the new leader for game %v!", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to FILE_UPLOAD success")
	}

	wasLeader := leader
	leader = tobeLeader

	wasLeader.conn = testGame.server.Connect(t)
	wasLeader.SendHello(t)
	resp = wasLeader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is FULL.", wasLeader.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected leader in FILE_UPLOAD")
	}

	wasLeader.SendStartGame(t, testGame.tag)
	badResp := wasLeader.ReadResponse(t)
	expectedErrorResponse := fmt.Sprintf("Only the leader can start the game. Please contact %v.", leader.name)
	if badResp != expectedErrorResponse {
		t.Fatalf("Incorrect response to START_GAME by previous leader in FILE_UPLOAD")
	}

	testGame.server.CleanUp(t)

	testGame = NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	leader = testGame.players[0]
	player := testGame.players[1]
	leader.SendStartGame(t, testGame.tag)

	expectedResponse = fmt.Sprintf("Game %s has started. Waiting for %s to upload the file.", testGame.tag, leader.name)
	resp = player.ReadResponse(t)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response after game started in FILE_UPLOAD")
	}

	expectedResponse = fmt.Sprintf("Game %v is running. Please upload the file.", testGame.tag)
	resp = leader.ReadResponse(t)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to leader after game started in FILE_UPLOAD")
	}

	testGame.GetFileSize(t)

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = player.ReadResponse(t)
	expectedErrorResponse = "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to reconnected player in FILE_UPLOAD")
	}

	player.SendHello(t)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is RUNNING.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player in FILE_UPLOAD")
	}

	player.SendFileUpload(t, "BAD", testGame.fileName, testGame.fileSize)
	resp = player.ReadResponse(t)
	expectedResponse = "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to FILE_UPLOAD with invalid tag")
	}

	player.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Only the leader can upload the file. Please contact %v.", leader.name)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to FILE_UPLOAD by non-leader")
	}

	leader.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = leader.ReadResponse(t)
	expectedResponse = "Upload completed! Waiting for word selection."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to player after FILE_UPLOAD")
	}

	leader.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = leader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Upload failed! File %v already exists for game %s.", testGame.fileName, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to FILE_UPLOAD with duplicate file")
	}

	testGame.server.CleanUp(t)
}

func TestFinal_RandomWord(t *testing.T) {
	testGame := NewTestGame(t, 5)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)
	testGame.StartGame(t)

	// Multi game support
	secondGame := &TestGame{
		players:     make([]*TestPlayer, 0),
		server:      testGame.server,
		playerCount: 2,
		fileName:    "test.txt",
		fileSize:    0,
	}
	for i := 0; i < secondGame.playerCount; i++ {
		secPlayer := NewPlayer(t, secondGame.server, 99+i)
		secondGame.players = append(secondGame.players, secPlayer)
		secPlayer.SendHello(t)
		resp := secPlayer.ReadResponse(t)
		expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Do you want to create a new game or join an existing game?", secPlayer.name)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to HELLO in second game")
		}
	}
	secondGame.NewGame(t)
	secondGame.JoinGame(t)
	secPlayer := secondGame.players[1]
	secPlayer.Close()
	secPlayer.conn = testGame.server.Connect(t)
	secPlayer.SendHello(t)
	resp := secPlayer.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is WAITING.", secPlayer.name, secondGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player in second game")
	}

	leader := testGame.players[0]
	player := testGame.players[4]

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendRandomWord(t, testGame.tag, "random")
	resp = player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to RANDOM_WORD before HELLO")
	}
	testGame.playerCount--

	testGame.GetFileSize(t)
	leader.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = leader.ReadResponse(t)
	expectedResponse = "Upload completed! Waiting for word selection."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to leader after FILE_UPLOAD")
	}

	playerCount := 1 // Leader
	pickerCount := 0
	pickerResponse := fmt.Sprintf("Upload completed! Please select a word from %s.", testGame.fileName)

	for j := 1; j < testGame.playerCount; j++ {
		p := testGame.players[j]
		time.Sleep(10 * time.Millisecond)
		playerResp := p.ReadResponse(t)
		if playerResp == expectedResponse {
			playerCount++
		} else if playerResp == pickerResponse {
			pickerCount++
			testGame.picker = p
		}
	}

	if pickerCount != 1 || playerCount != testGame.playerCount-1 {
		t.Fatalf("Incorrect count of players and picker after FILE_UPLOAD")
	}

	player.SendHello(t)
	testGame.playerCount++
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is RUNNING.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player after FILE_UPLOAD")
	}

	player.SendRandomWord(t, "BAD", "random")
	resp = player.ReadResponse(t)
	expectedResponse = "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to RANDOM_WORD with invalid tag")
	}

	player.SendRandomWord(t, testGame.tag, "random")
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Only the picker can pick the word. Please contact %v.", testGame.picker.name)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to RANDOM_WORD by non-picker")
	}

	picker := testGame.picker
	picker.SendRandomWord(t, testGame.tag, "BAD")
	resp = picker.ReadResponse(t)
	expectedResponse = "Word BAD is not a valid choice, choose another word."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to RANDOM_WORD with invalid word")
	}

	picker.SendRandomWord(t, testGame.tag, "thy")
	expectedResponse = "Word selected is thy! Guess the word count."
	for _, player := range testGame.players {
		resp = player.ReadResponse(t)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to player after RANDOM_WORD selection")
		}
	}

	testGame.server.CleanUp(t)
}

func TestFinal_GuessCount(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)

	leader := testGame.players[0]
	leader.SendInvalid(t)
	resp := leader.ReadResponse(t)
	expectedResponse := "Error! Please send a valid command."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to invalid command")
	}

	testGame.server.CleanUp(t)

	testGame = NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)
	testGame.StartGame(t)

	leader = testGame.players[0]
	player := testGame.players[1]

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendGuessCount(t, testGame.tag, 0)
	resp = player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to WORD_COUNT before HELLO")
	}

	player.SendHello(t)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is RUNNING.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player in WORD_COUNT")
	}

	testGame.GetFileSize(t)
	leader.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = leader.ReadResponse(t)
	expectedResponse = "Upload completed! Waiting for word selection."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to leader after FILE_UPLOAD success")
	}

	playerCount := 1 // Leader
	pickerCount := 0
	pickerResponse := fmt.Sprintf("Upload completed! Please select a word from %s.", testGame.fileName)

	for j := 1; j < testGame.playerCount; j++ {
		p := testGame.players[j]
		time.Sleep(10 * time.Millisecond)
		playerResp := p.ReadResponse(t)
		if playerResp == expectedResponse {
			playerCount++
		} else if playerResp == pickerResponse {
			pickerCount++
			testGame.picker = p
		}
	}

	if pickerCount != 1 || playerCount != testGame.playerCount-1 {
		t.Fatalf("Incorrect count of picker/players after FILE_UPLOAD")
	}

	player.SendGuessCount(t, "BAD", 0)
	resp = player.ReadResponse(t)
	expectedResponse = "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to WORD_COUNT with invalid tag")
	}

	player.SendGuessCount(t, testGame.tag, 0)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("No word has been selected yet for game %v. Wait!", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to WORD_COUNT before RANDOM_WORD")
	}

	picker := testGame.picker
	picker.SendRandomWord(t, testGame.tag, "thy")
	expectedResponse = "Word selected is thy! Guess the word count."

	winnerCount, loserCount := 0, 0

	for _, p := range testGame.players {
		resp = p.ReadResponse(t)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to RANDOM_WORD selection")
		}
		selectedGuess := rand.Intn(100)
		p.SendGuessCount(t, testGame.tag, (int64)(selectedGuess))
	}

	loserResp := "Sorry you lose! Better luck next time."
	winnerResp := "Congratulations you are the winner!"

	for _, p := range testGame.players {
		resp = p.ReadResponse(t)
		if resp == loserResp {
			loserCount++
		} else if resp == winnerResp {
			winnerCount++
		}
	}

	if winnerCount != 1 || loserCount != testGame.playerCount-1 {
		t.Fatalf("Incorrect count of players after WORD_COUNT")
	}

	resp = leader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Game %s complete. Do you want to restart or close the game?", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to leader after game is complete")
	}

	testGame.server.CleanUp(t)

}

func TestFinal_Restart(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	leader := testGame.players[0]
	player := testGame.players[1]

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendRestart(t, testGame.tag)
	resp := player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to reconnected player without HELLO in RESTART")
	}

	player.SendHello(t)
	resp = player.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is FULL.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player with HELLO in RESTART")
	}

	player.SendRestart(t, "BAD")
	resp = player.ReadResponse(t)
	expectedResponse = "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to RESTART with invalid tag")
	}

	player.SendRestart(t, testGame.tag)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Only the leader can restart the game. Please contact %v.", leader.name)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to RESTART by non-leader")
	}

	leader.SendRestart(t, testGame.tag)
	expectedResponse = "New game started!"

	for _, p := range testGame.players {
		resp = p.ReadResponse(t)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to player after RESTART by leader")
		}
	}

	leader.SendJoinGame(t, testGame.tag)
	resp = leader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Game %s is full or already in progress. Connect back later.", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to JOIN_GAME after game RESTART")
	}

	testGame.server.CleanUp(t)
}

func TestFinal_Close(t *testing.T) {
	testGame := NewTestGame(t, 8)
	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	leader := testGame.players[0]
	player := testGame.players[1]

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendClose(t, testGame.tag)
	resp := player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to reconnected player CLOSE without HELLO")
	}

	player.SendHello(t)
	resp = player.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is FULL.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player CLOSE with HELLO")
	}

	player.SendClose(t, "BAD")
	resp = player.ReadResponse(t)
	expectedResponse = "Game BAD doesn't exist! Please enter correct tag or create a new game."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to CLOSE with invalid tag")
	}

	player.SendClose(t, testGame.tag)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Only the leader can close the game. Please contact %v.", leader.name)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to CLOSE by non-leader")
	}

	leader.SendClose(t, testGame.tag)
	expectedResponse = "Bye!"

	for _, p := range testGame.players {
		resp = p.ReadResponse(t)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to CLOSE by leader")
		}
	}

	leader.SendJoinGame(t, testGame.tag)
	resp = leader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Game %s doesn't exist! Please enter correct tag or create a new game.", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to JOIN_GAME after CLOSE")
	}

	testGame.server.CleanUp(t)
}

func TestFinal_Goodbye(t *testing.T) {
	testGame := NewTestGame(t, 8)

	tempPlayer := NewPlayer(t, testGame.server, 0)
	tempPlayer.SendHello(t)
	resp := tempPlayer.ReadResponse(t)
	expectedResponse := fmt.Sprintf("Welcome to Word Count %s! Do you want to create a new game or join an existing game?", tempPlayer.name)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to valid HELLO in GOODBYE")
	}

	tempPlayer.SendGoodbye(t)
	expectedResponse = "Bye!"
	resp = tempPlayer.ReadResponse(t)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to valid GOODBYE")
	}
	tempPlayer.Close()

	testGame.GameSetup(t)
	testGame.NewGame(t)
	testGame.JoinGame(t)

	leader := testGame.players[0]
	player := testGame.players[4]

	player.Close()
	player.conn = testGame.server.Connect(t)
	player.SendGoodbye(t)
	resp = player.ReadResponse(t)
	expectedErrorResponse := "New player must always start with HELLO!"
	if resp != expectedErrorResponse {
		t.Fatalf("Incorrect response to reconnected player GOODBYE without HELLO")
	}
	player.Close()

	player.conn = testGame.server.Connect(t)
	player.SendHello(t)
	resp = player.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is FULL.", player.name, testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player HELLO in GOODBYE")
	}

	player.SendGoodbye(t)
	resp = player.ReadResponse(t)
	expectedResponse = "Bye!"
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to reconnected player GOODBYE")
	}

	leader.SendGoodbye(t)
	expectedResponse = "Bye!"
	for _, player := range testGame.players {
		time.Sleep(10 * time.Millisecond)
		resp = player.ReadResponse(t)
		if resp != expectedResponse {
			t.Fatalf("Incorrect response to leader GOODBYE")
		}
	}

	os.MkdirAll(RootDir+StorageDirectoryName, os.ModePerm) // Recreate directory
	leader.SendJoinGame(t, testGame.tag)
	resp = leader.ReadResponse(t)
	expectedResponse = fmt.Sprintf("Game %s doesn't exist! Please enter correct tag or create a new game.", testGame.tag)
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to JOIN_GAME after leader GOODBYE")
	}

	testGame.NewGame(t)
	testGame.JoinGame(t)
	testGame.StartGame(t)
	testGame.GetFileSize(t)
	leader.SendFileUpload(t, testGame.tag, testGame.fileName, testGame.fileSize)
	resp = leader.ReadResponse(t)
	expectedResponse = "Upload completed! Waiting for word selection."
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to FILE_UPLOAD in GOODBYE")
	}

	playerCount := 1 // Leader
	pickerCount := 0
	pickerResponse := fmt.Sprintf("Upload completed! Please select a word from %s.", testGame.fileName)

	for j := 0; j < testGame.playerCount; j++ {
		p := testGame.players[j]
		if p == leader {
			continue
		}
		time.Sleep(10 * time.Millisecond)
		playerResp := p.ReadResponse(t)
		if playerResp == expectedResponse {
			playerCount++
		} else if playerResp == pickerResponse {
			pickerCount++
			testGame.picker = p
		}
	}

	if pickerCount != 1 || playerCount != testGame.playerCount-1 {
		t.Fatalf("Incorrect picker/players count after FILE_UPLOAD in GOODBYE")
	}

	testGame.picker.SendGoodbye(t)
	resp = testGame.picker.ReadResponse(t)
	expectedResponse = "Bye!"
	if resp != expectedResponse {
		t.Fatalf("Incorrect response to GOODBYE from picker")
	}

	testGame.server.CleanUp(t)
}
