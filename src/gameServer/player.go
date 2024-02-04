package main

import (
	"bufio"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func readFile(chanInput chan string, cmdLine string) string {
	var builder strings.Builder
	cmd := strings.Split(cmdLine, " ")
	fileSize, _ := strconv.Atoi(cmd[3])
	start := strings.Index(cmdLine, cmd[3]) + len(cmd[3]) + 1
	builder.WriteString(cmdLine[start:])
	builder.WriteString("\n")
	fileSize -= len(cmdLine) - start + 1
	for fileSize > 0 {
		line := <-chanInput
		builder.WriteString(line)
		builder.WriteString("\n")
		fileSize -= len(line) + 1
	}
	<-chanInput
	return builder.String()
}

type Player struct {
	name    string
	gameIDs map[string]chan map[string]string // joined games and their mailboxes
	mailbox chan map[string]string
	server  *GameServer
}

// client routine
func clientRoutine(conn net.Conn, server *GameServer) error {
	scanner := bufio.NewScanner(conn)
	var player *Player
	leaders := make(map[string]string)
	chanInput := make(chan string)
	go func() {
		for scanner.Scan() {
			chanInput <- scanner.Text()
		}
		close(chanInput)
	}()

	// hello
	hello := false
	for cmdLine := range chanInput {
		cmd := strings.Split(cmdLine, " ")
		if cmd[0] == "FILE_UPLOAD" {
			readFile(chanInput, cmdLine)
		}
		if cmd[0] != "HELLO" {
			io.WriteString(conn, msgNoHello())
			continue
		}
		if len(cmd) != 2 {
			io.WriteString(conn, msgInvalidArgs("HELLO"))
			continue
		}
		if len(cmd[1]) == 0 {
			io.WriteString(conn, msgInvalidUsrname())
			continue
		}
		username := cmd[1]
		server.chanName <- username
		player = <-server.chanPlayer
		hello = true
		break
	}
	if !hello {
		conn.Close()
		return nil
	}
	if len(player.gameIDs) == 0 {
		io.WriteString(conn, msgWelcome(player.name, "", ""))
	} else {
		for gameID, gameChannel := range player.gameIDs {
			infoRequest := map[string]string{
				"cmd":  "RECONN",
				"name": player.name,
			}
			gameChannel <- infoRequest

			response := <-player.mailbox
			if response["status"] != "success" {
				// fails to reconnect
				conn.Close()
				return nil
			}

			leader := response["leader"]
			leaders[gameID] = leader

			io.WriteString(conn, msgWelcome(player.name, gameID, response["state"]))
			break
		}
	}

	disconn := false
loop:
	for {
		select {
		case cmdLine, more := <-chanInput:
			if !more {
				disconn = true
				break loop
			}
			cmd := strings.Split(cmdLine, " ")
			switch cmd[0] {
			case "NEW_GAME":
				if len(cmd) != 2 {
					io.WriteString(conn, msgInvalidArgs("NEW_GAME"))
					continue
				}
				req := struct {
					gameID  string
					name    string
					newGame bool
				}{
					gameID:  cmd[1],
					name:    player.name,
					newGame: true,
				}
				server.chanGameReq <- req
				game := <-server.chanGameResp
				if game == nil {
					io.WriteString(conn, msgGameExists(cmd[1]))
					continue
				}
				io.WriteString(conn, msgGameCreated(cmd[1]))
				player.gameIDs[cmd[1]] = game // add to joined games map
				leaders[cmd[1]] = player.name

			case "JOIN_GAME":
				// Check if the command has the correct number of arguments
				if len(cmd) != 2 {
					io.WriteString(conn, msgInvalidArgs("JOIN_GAME"))
					continue
				}
				// Prepare a join game request
				req := struct {
					gameID  string
					name    string
					newGame bool
				}{
					gameID:  cmd[1], // Game ID to join
					name:    player.name,
					newGame: false, // Indicates this is a join request, not a new game request
				}

				// Send join game request to server
				server.chanGameReq <- req
				game := <-server.chanGameResp

				// Check if the game was successfully joined
				if game == nil {
					io.WriteString(conn, msgGameNotFound(cmd[1]))
					continue
				}

				request := map[string]string{
					"cmd":  "JOIN",
					"name": player.name,
				}

				// Send the message string to the game's logic
				game <- request

				// Wait for a response in the player's mailbox
				response := <-player.mailbox

				// Process the response
				status := response["status"]

				if status == "success" {
					// Update player's gameIDs map to include the joined game
					player.gameIDs[cmd[1]] = game
					io.WriteString(conn, msgGameJoined(cmd[1], response["state"]))
					leaders[cmd[1]] = response["leader"]
				} else {
					io.WriteString(conn, msgJoinGameFail(cmd[1]))
				}

			case "START_GAME":
				// Check if the command has the correct number of arguments
				if len(cmd) != 2 {
					io.WriteString(conn, msgInvalidArgs("START_GAME"))
					continue
				}

				gameID := cmd[1]
				// Check if the player is in the game
				game, ok := player.gameIDs[gameID]

				if !ok {
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  cmd[1], // Game ID to join
						name:    player.name,
						newGame: false, // Indicates this is a join request, not a new game request
					}
					server.chanGameReq <- req
					game = <-server.chanGameResp
					if game == nil {
						io.WriteString(conn, msgGameNotFound(cmd[1]))
						continue
					}
				}

				// Prepare a start game request
				request := map[string]string{
					"cmd":    "START",
					"gameID": gameID,
					"name":   player.name,
				}

				// Send the start game request to the game's logic
				game <- request

				// Wait for a response in the player's mailbox
				response := <-player.mailbox

				// Process the response
				status := response["status"]
				if status == "success" {
					io.WriteString(conn, msgGameStartedLeader(gameID))
				} else {
					// Handle failure to start game
					reason := response["reason"]
					wait := response["wait"]
					leader := response["leader"]
					io.WriteString(conn, msgStartGameFail(gameID, reason, wait, leader))
				}

			case "FILE_UPLOAD":
				gameID := cmd[1]
				fileName := cmd[2]
				fileData := readFile(chanInput, cmdLine)
				leader, ok := leaders[gameID]
				if !ok {
					// did not join the game
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  gameID, // Game ID to join
						name:    player.name,
						newGame: false, // Indicates this is a join request, not a new game request
					}
					server.chanGameReq <- req
					mailbox := <-server.chanGameResp
					if mailbox == nil {
						io.WriteString(conn, msgGameNotFound(gameID))
						continue
					}
					request := map[string]string{"cmd": "INFO", "name": player.name}
					mailbox <- request
					response := <-player.mailbox
					io.WriteString(conn, msgNonLeaderUpload(response["leader"]))
					continue
				}
				if leader != player.name {
					// joined the game but I am not the leader
					io.WriteString(conn, msgNonLeaderUpload(leader))
					continue
				}
				request := map[string]string{"cmd": "UPLOAD", "name": player.name, "filename": fileName}
				mailbox := player.gameIDs[gameID]
				mailbox <- request
				response := <-player.mailbox
				if response["status"] == "fail" {
					// a file with the same name exists
					io.WriteString(conn, msgFileExists(gameID, fileName))
					continue
				}
				f, err := os.Create(response["path"] + fileName)
				if err != nil {
					// unable to create the file, return a fail to the game
					mailbox <- map[string]string{"status": "fail"}
					continue
				}
				f.WriteString(fileData)
				f.Close()
				// tell the game the upload is complete
				mailbox <- map[string]string{"status": "success"}
				// do not print anything here, wait for the server's notification

			case "RANDOM_WORD":
				if len(cmd) < 3 {
					io.WriteString(conn, msgInvalidArgs("RANDOM_WORD"))
					continue
				}

				gameID, word := cmd[1], cmd[2]

				// The game to which the word is being set
				game, ok := player.gameIDs[gameID]
				if !ok {
					// Game not found in player's current games, request it from the server
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  gameID,
						name:    player.name,
						newGame: false,
					}
					server.chanGameReq <- req
					game = <-server.chanGameResp

					if game == nil {
						// Game does not exist
						io.WriteString(conn, msgGameNotFound(gameID))
						continue
					} else {
						io.WriteString(conn, msgInvalidCmd())
						continue
					}
				}

				// Send the chosen word to the game logic
				wordRequest := map[string]string{
					"cmd":  "RANDOM_WORD",
					"name": player.name,
					"word": word,
				}
				game <- wordRequest

				// Wait for a response from the game logic
				response := <-player.mailbox

				// Handle the response
				if response["status"] != "success" {
					reason := response["reason"]
					picker := response["picker"]
					io.WriteString(conn, msgWordSetFail(reason, word, picker, response["leader"]))
				}

			case "WORD_COUNT":
				if len(cmd) != 3 {
					io.WriteString(conn, msgInvalidArgs("WORD_COUNT"))
					continue
				}

				gameID, guess := cmd[1], cmd[2]

				// Check if the player is part of the specified game
				game, ok := player.gameIDs[gameID]
				if !ok {
					// Player is not part of the game, request game info from server
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  gameID,
						name:    player.name,
						newGame: false,
					}
					server.chanGameReq <- req
					game = <-server.chanGameResp

					if game == nil {
						io.WriteString(conn, msgGameNotFound(cmd[1]))
						continue
					}
				}

				// Send the guess to the game logic
				guessRequest := map[string]string{
					"cmd":   "WORD_COUNT",
					"name":  player.name,
					"guess": guess,
				}
				game <- guessRequest

				// Wait for a response from the game logic
				response := <-player.mailbox

				// Handle the response
				if response["status"] != "success" {
					reason := response["reason"]
					io.WriteString(conn, msgWordCountFail(reason, cmd[1]))
				}

			case "RESTART":
				if len(cmd) != 2 {
					io.WriteString(conn, msgInvalidArgs("RESTART"))
					continue
				}

				gameID := cmd[1]

				// Check if the player is part of the specified game
				game, ok := player.gameIDs[gameID]
				if !ok {
					// Player is not part of the game, request game info from server
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  gameID,
						name:    player.name,
						newGame: false,
					}
					server.chanGameReq <- req
					game = <-server.chanGameResp

					if game == nil {
						io.WriteString(conn, msgGameNotFound(cmd[1]))
						continue
					}
				}

				// Send the restart command to the game logic
				restartRequest := map[string]string{
					"cmd":  "RESTART",
					"name": player.name,
				}
				game <- restartRequest

				// Wait for a response from the game logic
				response := <-player.mailbox

				// Handle the response
				if response["status"] != "success" {
					io.WriteString(conn, msgGameRestartFail(leaders[gameID]))
				}

			case "CLOSE":
				if len(cmd) != 2 {
					io.WriteString(conn, msgInvalidArgs("CLOSE"))
					continue
				}

				gameID := cmd[1]

				// Check if the player is the leader of the specified game
				game, ok := player.gameIDs[gameID]
				if !ok {
					// Player is not part of the game, request game info from server
					req := struct {
						gameID  string
						name    string
						newGame bool
					}{
						gameID:  gameID,
						name:    player.name,
						newGame: false,
					}
					server.chanGameReq <- req
					game = <-server.chanGameResp

					if game == nil {
						io.WriteString(conn, msgGameNotFound(cmd[1]))
						continue
					}
				}

				// Send the close command to the game logic
				closeRequest := map[string]string{
					"cmd":    "CLOSE",
					"gameID": gameID,
					"name":   player.name,
				}
				game <- closeRequest

				// Wait for a response from the game logic
				response := <-player.mailbox

				// Handle the response
				if response["status"] != "success" {
					io.WriteString(conn, msgGameCloseFail(leaders[gameID]))
				}
				// clear info about the game
				delete(player.gameIDs, gameID)
				delete(leaders, gameID)

			case "GOODBYE":
				for gameID, gameChannel := range player.gameIDs {
					closeRequest := map[string]string{
						"cmd":    "GOODBYE",
						"gameID": gameID,
						"name":   player.name,
					}
					gameChannel <- closeRequest
					<-player.mailbox
				}
				io.WriteString(conn, msgBye())

			default:
				io.WriteString(conn, msgInvalidCmd())
			}

		case notification := <-player.mailbox:
			switch notification["msg"] {
			case "READY":
				io.WriteString(conn, msgGameReady(notification["gameID"]))
			case "STARTED":
				io.WriteString(conn, msgGameStartedNonLeader(notification["gameID"], notification["leader"]))
			case "UPLOADED":
				io.WriteString(conn, msgFileUploadedNonPicker())
			case "PICK":
				io.WriteString(conn, msgFileUploadedPicker(notification["filename"]))
			case "NEW_LEADER":
				gameID := notification["gameID"]
				leader := notification["leader"]
				leaders[gameID] = leader
				if player.name == leader {
					io.WriteString(conn, msgBecomeNewLeader(gameID))
				}
			case "WORD_SELECTED":
				io.WriteString(conn, msgWordSetSuccess(notification["word"]))
			case "WINNER":
				gameID := notification["gameID"]
				if player.name == notification["name"] {
					io.WriteString(conn, msgIsWinner())
				} else {
					io.WriteString(conn, msgIsLoser())
				}
				time.Sleep(1 * time.Second) // make the test happy
				if leaders[gameID] == player.name {
					io.WriteString(conn, msgRestartOrClose(gameID))
				}
			case "RESTARTED":
				io.WriteString(conn, msgGameRestarted())
			case "CLOSED":
				gameID := notification["gameID"]
				io.WriteString(conn, msgBye())
				delete(player.gameIDs, gameID)
				delete(leaders, gameID)
			case "EXIT":
				gameID := notification["gameID"]
				delete(player.gameIDs, gameID)
				delete(leaders, gameID)
				break loop
			}
		}
	}

	if disconn {
		// disconnected, tell the game
		request := map[string]string{"cmd": "DISCONN", "name": player.name}
		for _, mailbox := range player.gameIDs {
			mailbox <- request
		}
	} else {
		// exit
		for len(player.gameIDs) > 0 {
			notification := <-player.mailbox
			gameID := notification["gameID"]
			delete(player.gameIDs, gameID)
			delete(leaders, gameID)
		}
		conn.Close()
		close(player.mailbox)
	}

	return nil
}
