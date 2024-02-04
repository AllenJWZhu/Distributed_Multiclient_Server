package main

import "fmt"

// normal status messages

func msgWelcome(username string, gameID string, gameState string) string {
	if gameState == "" {
		// new player
		return fmt.Sprintf("Welcome to Word Count %s! Do you want to create a new game or join an existing game?\n", username)
	}
	return fmt.Sprintf("Welcome to Word Count %s! Resumed Game %s. Current state is %s.\n", username, gameID, gameState)
}

func msgGameCreated(gameID string) string {
	return fmt.Sprintf("Game %s created! You are the leader of the game. Waiting for players to join.\n", gameID)
}

func msgGameJoined(gameID string, state string) string {
	return fmt.Sprintf("Joined Game %s. Current state is %s.\n", gameID, state)
}

func msgGameReady(gameID string) string {
	return fmt.Sprintf("Game %s is ready to start.\n", gameID)
}

func msgGameStartedLeader(gameID string) string {
	return fmt.Sprintf("Game %s is running. Please upload the file.\n", gameID)
}

func msgGameStartedNonLeader(gameID string, leader string) string {
	return fmt.Sprintf("Game %s has started. Waiting for %s to upload the file.\n", gameID, leader)
}

func msgFileUploadedNonPicker() string {
	return "Upload completed! Waiting for word selection.\n"
}

func msgFileUploadedPicker(fileName string) string {
	return fmt.Sprintf("Upload completed! Please select a word from %s.\n", fileName)
}

func msgBecomeNewLeader(gameID string) string {
	return fmt.Sprintf("You are the new leader for game %s!\n", gameID)
}

func msgWordSetSuccess(word string) string {
	return fmt.Sprintf("Word selected is %s! Guess the word count.\n", word)
}

func msgIsWinner() string {
	return "Congratulations you are the winner!\n"
}

func msgIsLoser() string {
	return "Sorry you lose! Better luck next time.\n"
}

func msgRestartOrClose(gameID string) string {
	return fmt.Sprintf("Game %s complete. Do you want to restart or close the game?\n", gameID)
}

func msgGameRestarted() string {
	return "New game started!\n"
}

func msgBye() string {
	return "Bye!\n"
}

// errors

func msgInvalidArgs(cmd string) string {
	return fmt.Sprintf("Invalid arguments for command %s.\n", cmd)
}

func msgNoHello() string {
	return "New player must always start with HELLO!\n"
}

func msgInvalidUsrname() string {
	return "Invalid user name. Try again.\n"
}

func msgGameExists(gameID string) string {
	return fmt.Sprintf("Game %s already exists, please provide a new game tag.\n", gameID)
}

func msgGameNotFound(gameID string) string {
	return fmt.Sprintf("Game %s doesn't exist! Please enter correct tag or create a new game.\n", gameID)
}

func msgJoinGameFail(gameID string) string {
	return fmt.Sprintf("Game %s is full or already in progress. Connect back later.\n", gameID)
}

func msgStartGameFail(gameID, reason, wait, leader string) string {
	switch reason {
	case "already started":
		return fmt.Sprintf("Game %s has already started! Please create a new game.\n", gameID)
	case "not a leader":
		return fmt.Sprintf("Only the leader can start the game. Please contact %s.\n", leader)
	case "not enough players":
		return fmt.Sprintf("Can't start the game %s, waiting for %s more players.\n", gameID, wait)
	default:
		return "An unknown error occurred while attempting to start the game.\n"
	}
}

func msgNonLeaderUpload(leader string) string {
	return fmt.Sprintf("Only the leader can upload the file. Please contact %s.\n", leader)
}

func msgFileExists(gameID string, fileName string) string {
	return fmt.Sprintf("Upload failed! File %s already exists for game %s.\n", fileName, gameID)
}

func msgWordSetFail(reason, word string, pickerName string, leader string) string {
	switch reason {
	case "not a picker":
		return fmt.Sprintf("Only the picker can pick the word. Please contact %s.\n", pickerName)
	case "not a valid choice":
		return fmt.Sprintf("Word %s is not a valid choice, choose another word.\n", word)
	case "file not ready":
		return fmt.Sprintf("No file uploaded. Please contact %s.\n", leader)
	default:
		return "An unknown error occurred while attempting to set the word.\n"
	}
}

func msgInvalidCmd() string {
	return "Error! Please send a valid command.\n"
}

func msgWordCountFail(reason, gameID string) string {
	switch reason {
	case "did not join the game":
		return "Error! Please send a valid command.\n"
	case "not ready for guesses":
		return fmt.Sprintf("No word has been selected yet for game %s. Wait!\n", gameID)
	default:
		return "An unknown error occurred while attempting to set the word.\n"
	}
}

func msgGameRestartFail(leader string) string {
	return fmt.Sprintf("Only the leader can restart the game. Please contact %s.\n", leader)
}

func msgGameCloseFail(leader string) string {
	return fmt.Sprintf("Only the leader can close the game. Please contact %s.\n", leader)
}
