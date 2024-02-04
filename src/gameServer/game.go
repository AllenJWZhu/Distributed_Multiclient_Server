package main

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

type GameState string

const (
	WAITING GameState = "WAITING"
	FULL    GameState = "FULL"
	READY   GameState = "READY"
	RUNNING GameState = "RUNNING"
)

type Game struct {
	gameID string
	state  GameState

	leader          string
	picker          string // who picks the word
	wordDict        map[string]int
	fileName        string
	tgtWord         string // target word
	usedWords       map[string]bool
	waitingForGuess bool           // Flag to indicate if the game is ready for guessing
	guessResults    map[string]int // To store player's guess results

	names        map[string]chan map[string]string // players in this game and their mailboxes
	namesDisconn map[string]chan map[string]string // players that lose connections
	namesBye     map[string]chan map[string]string // players that have said goodbye
	namesOrd     map[string]int                    // the order in which players join the game
	mailbox      chan map[string]string

	directory string
	exit      chan bool // force exit channel
	server    *GameServer
}

func (game *Game) routine() {
loop:
	for {
		select {
		case mail := <-game.mailbox:
			switch mail["cmd"] {
			case "JOIN":
				name := mail["name"]
				mailbox, ok := game.names[name]
				if ok {
					// this player has already joined
					response := map[string]string{"status": "fail"}
					mailbox <- response
					continue
				}
				// find this player's mailbox
				game.server.chanPlayerReq <- name
				mailbox = <-game.server.chanPlayerResp
				if game.state == WAITING || game.state == READY {
					// ok to join
					game.names[name] = mailbox
					game.namesOrd[name] = len(game.namesOrd)
					game.changeState()
					response := map[string]string{"status": "success", "state": string(game.state), "leader": game.leader}
					mailbox <- response
					if len(game.names) == MIN_PLAYERS {
						// notify the leader that the game is ready to start
						notification := map[string]string{
							"gameID": game.gameID,
							"msg":    "READY",
						}
						game.names[game.leader] <- notification
					}
				} else {
					// unable to join
					response := map[string]string{"status": "fail"}
					mailbox <- response
				}

			case "START":
				name := mail["name"]
				mailbox, ok := game.names[name]
				if !ok {
					game.server.chanPlayerReq <- name
					mailbox = <-game.server.chanPlayerResp
				}
				if name != game.leader {
					// non-leader issues START
					response := map[string]string{
						"status": "fail",
						"reason": "not a leader",
						"leader": game.leader,
					}
					mailbox <- response
					continue
				}
				if game.state == RUNNING {
					// game has started
					response := map[string]string{
						"status": "fail",
						"reason": "already started",
					}
					mailbox <- response
					continue
				}
				if game.state == WAITING {
					// not enough people
					response := map[string]string{
						"status": "fail",
						"reason": "not enough players",
						"wait":   strconv.Itoa(MIN_PLAYERS - len(game.names)),
					}
					mailbox <- response
					continue
				}
				// start the game
				game.state = RUNNING
				response := map[string]string{"status": "success"}
				mailbox <- response
				// tell every non-leader players
				notification := map[string]string{
					"gameID": game.gameID,
					"msg":    "STARTED",
					"leader": game.leader,
				}
				for k, v := range game.names {
					if k != game.leader {
						v <- notification
					}
				}

			case "UPLOAD":
				name := mail["name"]
				fileName := mail["filename"]
				// is this player the leader?
				if name != game.leader {
					response := map[string]string{"status": "fail"}
					mailbox, ok := game.names[name]
					if !ok {
						// the player did not join the game
						game.server.chanPlayerReq <- name
						mailbox = <-game.server.chanPlayerResp
					}
					mailbox <- response
					continue
				}
				// check my directory to see if one file has the same name
				entries, err := os.ReadDir(game.directory)
				if err != nil {
					fmt.Printf("error: cannot open %s", game.directory)
					os.Exit(-1)
				}
				valid := true
				for _, entry := range entries {
					if entry.Name() == fileName {
						valid = false
						break
					}
				}
				mailbox := game.names[name]
				if !valid {
					// a file with the same name exists
					response := map[string]string{"status": "fail"}
					mailbox <- response
					continue
				}
				response := map[string]string{"status": "success", "path": game.directory}
				mailbox <- response
				uploadStatus := <-game.mailbox
				if uploadStatus["status"] != "success" {
					continue
				}
				game.fileName = fileName
				// choose a picker
				names := make([]string, 0, len(game.names))
				for name := range game.names {
					if name != game.leader {
						names = append(names, name)
					}
				}
				game.picker = names[rand.Intn(len(names))]
				// send a notification to the picker
				notification := map[string]string{"gameID": game.gameID, "msg": "PICK", "filename": fileName}
				game.names[game.picker] <- notification
				// send a notification to everyone else
				msg := map[string]string{"gameID": game.gameID, "msg": "UPLOADED"}
				for name, mailbox := range game.names {
					if name != game.picker {
						mailbox <- msg
					}
				}
				// read the file, construct wordDict
				fd, err := os.Open(game.directory + fileName)
				if err != nil {
					fmt.Printf("error: cannot open %s", game.directory+fileName)
					os.Exit(-1)
				}
				scanner := bufio.NewScanner(fd)
				for scanner.Scan() {
					line := scanner.Text()
					if line == "" {
						continue
					}
					words := strings.Split(line, " ")
					for _, word := range words {
						_, ok := game.wordDict[word]
						if !ok {
							game.wordDict[word] = 1
						} else {
							game.wordDict[word]++
						}
					}
				}
				fd.Close()

			case "RANDOM_WORD":
				name := mail["name"]
				word := mail["word"]
				if name != game.picker {
					// not the picker
					mailbox, ok := game.names[name]
					if !ok {
						// the player did not join the game
						game.server.chanPlayerReq <- name
						mailbox = <-game.server.chanPlayerResp
					}
					resp := map[string]string{"status": "fail", "reason": "not a picker", "picker": game.picker}
					mailbox <- resp
					continue
				}
				mailbox := game.names[name]
				if len(game.directory) == 0 {
					// file not yet uploaded
					resp := map[string]string{"status": "fail", "reason": "file not ready"}
					mailbox <- resp
					continue
				}
				_, inDict := game.wordDict[word]
				_, used := game.usedWords[word]
				if !inDict || used {
					// the word is not in the file or has been used
					resp := map[string]string{"status": "fail", "reason": "not a valid choice"}
					mailbox <- resp
					continue
				}
				// successfully uploaded the word
				mailbox <- map[string]string{"status": "success"}
				game.tgtWord = word
				// notify everyone
				notification := map[string]string{"gameID": game.gameID, "msg": "WORD_SELECTED", "word": game.tgtWord}
				for _, box := range game.names {
					box <- notification
				}
				game.waitingForGuess = true // wait for players to submit their guesses

			case "WORD_COUNT":
				// Only process WORD_COUNT if the game is in the correct state
				name := mail["name"]
				mailbox, ok := game.names[name]
				if !ok {
					// the player did not join the game
					game.server.chanPlayerReq <- name
					mailbox = <-game.server.chanPlayerResp
					resp := map[string]string{"status": "fail", "reason": "did not join the game"}
					mailbox <- resp
					continue
				}
				if !game.waitingForGuess {
					resp := map[string]string{"status": "fail", "reason": "not ready for guesses"}
					mailbox <- resp
					continue
				}
				count := mail["guess"]
				guess, err := strconv.Atoi(count)

				if err != nil {
					// Handle invalid guess format
					resp := map[string]string{"status": "fail", "reason": "invalid format"}
					mailbox <- resp
					continue
				}

				// Record the player's guess
				game.guessResults[name] = guess
				// return success
				mailbox <- map[string]string{"status": "success"}

				// Check if all players have made their guesses
				if len(game.guessResults) >= len(game.names) {
					game.waitingForGuess = false
					winner := game.determineWinner(game.wordDict[game.tgtWord])
					for _, mailbox := range game.names {
						notification := map[string]string{
							"gameID": game.gameID,
							"msg":    "WINNER",
							"name":   winner,
						}
						mailbox <- notification
					}
				}

			case "DISCONN":
				name := mail["name"]
				game.namesDisconn[name] = game.names[name]
				delete(game.names, name)
				if game.state != RUNNING {
					game.changeState()
				}
				if name == game.leader {
					// select a new leader
					var newLeader string
					minOrd := len(game.names)
					for name := range game.names {
						ord := game.namesOrd[name]
						if ord < minOrd {
							minOrd = ord
							newLeader = name
						}
					}
					game.leader = newLeader
					msg := map[string]string{"gameID": game.gameID, "msg": "NEW_LEADER", "leader": game.leader}
					for _, mailbox := range game.names {
						mailbox <- msg
					}
				}

			case "RECONN":
				name := mail["name"]
				game.names[name] = game.namesDisconn[name]
				delete(game.namesDisconn, name)
				if game.state != RUNNING {
					game.changeState()
				}
				resp := map[string]string{"status": "success", "leader": game.leader, "state": string(game.state)}
				game.names[name] <- resp

			case "RESTART":
				name := mail["name"]
				mailbox, ok := game.names[name]
				if !ok {
					// the player did not join the game
					game.server.chanPlayerReq <- name
					mailbox = <-game.server.chanPlayerResp
				}
				if name != game.leader {
					mailbox <- map[string]string{"status": "fail"}
					continue
				}
				mailbox <- map[string]string{"status": "success"}
				// restart the game
				game.changeState()
				game.picker = ""
				if game.tgtWord != "" {
					game.usedWords[game.tgtWord] = true
					game.tgtWord = ""
				}
				game.waitingForGuess = false
				game.guessResults = make(map[string]int)
				// send notifications about the restart to everyone
				notification := map[string]string{
					"gameID": game.gameID,
					"msg":    "RESTARTED",
				}
				for _, box := range game.names {
					box <- notification
				}

			case "CLOSE":
				name := mail["name"]
				mailbox, ok := game.names[name]
				if !ok {
					// the player did not join the game
					game.server.chanPlayerReq <- name
					mailbox = <-game.server.chanPlayerResp
				}
				if name != game.leader {
					mailbox <- map[string]string{"status": "fail"}
					continue
				}
				game.cleanup(false)
				mailbox <- map[string]string{"status": "success"}
				// close the game, notify everyone
				notification := map[string]string{
					"gameID": game.gameID,
					"msg":    "CLOSED",
				}
				for _, box := range game.names {
					box <- notification
				}
				break loop

			case "GOODBYE":
				name := mail["name"]
				if name == game.leader {
					// close the game
					game.cleanup(false)
					game.names[name] <- map[string]string{"status": "success"}
					notification := map[string]string{
						"gameID": game.gameID,
						"msg":    "CLOSED",
					}
					for nm, box := range game.names {
						if nm != game.leader {
							box <- notification
						}
					}
					for _, box := range game.namesBye {
						box <- notification
					}
					break loop
				}
				game.names[name] <- map[string]string{"status": "success"}
				game.namesBye[name] = game.names[name]
				delete(game.names, name)
				if game.state == RUNNING {
					if name == game.picker && game.tgtWord == "" {
						// picker has not chosen the word, choose a new picker
						names := make([]string, 0, len(game.names))
						for name := range game.names {
							if name != game.leader {
								names = append(names, name)
							}
						}
						game.picker = names[rand.Intn(len(names))]
						// send a notification to the picker
						notification := map[string]string{
							"gameID":   game.gameID,
							"msg":      "PICK",
							"filename": game.fileName,
						}
						game.names[game.picker] <- notification
					}
				} else {
					game.changeState()
				}
			}
		case <-game.exit:
			game.cleanup(true)
			break loop
		}
	}
}

func (game *Game) cleanup(terminate bool) {
	os.RemoveAll(game.directory)
	if !terminate {
		game.server.chanGameExit <- game.gameID
		<-game.server.chanGameExitResp
	} else {
		// wait for every player to exit
		for _, mailbox := range game.names {
			mailbox <- map[string]string{"gameID": game.gameID, "msg": "EXIT"}
		}
		for _, mailbox := range game.namesBye {
			mailbox <- map[string]string{"gameID": game.gameID, "msg": "EXIT"}
		}
		game.exit <- true // confirm to server
	}
	close(game.mailbox)
}

func (game *Game) determineWinner(actualWordCount int) string {
	minDiff := math.MaxInt32
	var winner string

	for name, guess := range game.guessResults {
		diff := math.Abs(float64(guess - actualWordCount))
		if int(diff) < minDiff {
			minDiff = int(diff)
			winner = name
		}
	}

	return winner
}

func (game *Game) changeState() {
	if len(game.names) < MIN_PLAYERS {
		game.state = WAITING
	} else if len(game.names) < MAX_PLAYERS {
		game.state = READY
	} else {
		game.state = FULL
	}
}
