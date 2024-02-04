## Lab 0 - A Game about Words, Go version

This file details the contents of the initial Lab 0 code repository and how to use it.

### Getting started

This repository includes the Go game server and test code.  All you should need to complete this 
lab and run the tests is a recent Go compiler and a few simple utilities.  We're not using any libraries
outside of the stock Go SDK.  If you're setting up a basic environment for Go labs in this course, a default
Ubuntu Linux installation would only require you to do
```
sudo apt update; sudo apt -y upgrade
sudo apt install -y git make

# Install Go
sudo snap install go --classic
go env -w GO111MODULE=auto
```
to start working on the lab. Everything else is controlled by `make` using the `go` command and shell scripts. For
more details about installing `go`, see [https://go.dev/doc/install](https://go.dev/doc/install).


### Initial repository contents

The top-level directory (called `lab0-go` here) of the initial starter-code repository includes three things:
* This `README.md` file
* The Lab 0 `Makefile`, described in detail later
* The `src` source directory, which is where all your work will be done

Visually, this looks roughly like the following:
```
\---lab0-go
        +---src
        |   \---gameServer
        |   |   +---gameServer.go
        |   |   +---gameServer_test.go
        |   |   \---test.txt
        +---Makefile
        \---README.md
```
The details of each of these will hopefully become clear after reading the rest of this file.


### Creating your game server

The `gameServer` package initially only includes a skeleton of the game server implementation for Lab 0. Your
primary task in this lab is to complete this implementation according to the specifications given in the Canvas
assignment. You are free to write all the code in the single `gameServer.go` file or create additional files
as needed. The tests are performed using Go's built-in testing capabilities, so it may be helpful to get familiar
with the Go `testing` package, so you can see how the tests work and what they are testing.


### Testing the game server

Once you're at the point where you want to run any of the provided tests, you can use the provided `make` rules. To run
the set of Checkpoint tests, execute `make checkpoint` from the main working directory of the lab. Similarly, to run the 
Final tests, execute `make final`. To run all of the tests (Checkpoint and Final) at once, execute `make all`, which is 
provided for your convenience, as it is not used by our auto-grader. You can also run subsets of tests using the 
corresponding `go test` command arguments.

Before you're ready to run the tests, it's probably helpful to interact with your game server manually, where you are 
taking the roles of the game players. By far the easiest way to do this is to use the netcat (`nc`) utility, but there are
many options available to you.  Using netcat, all you need to do is start your game server in one terminal and connect to
it as players in other terminals.   For example, to run a game server on port `14736` and connect multiple players, execute
```
go run gameServer.go -port=localhost:14736
```
in one terminal and execute
```
nc localhost 14736
```
in a different terminal for each game player. Both of these commands initially hang with an empty terminal, and each player 
simply enters their text game commands into the terminal, pressing Enter/Return after each.  For example, the command 
`HELLO playerOne` introduces a new player to the game server, and the server should respond with 
`Welcome to Word Count playerOne! Do you want to create a new game or join an existing game?`, which will appear in the 
terminal. If you want your game server to print anything to its terminal (which is very helpful), you'll need to program 
it to do so.  However, as your game progresses, it might make sense to log server interactions to a file instead of the 
terminal, as there's really a lot of stuff going on in the fully working game.

If you're curious to see more details of the socket communication between your game server and your players, you can use network
utilities like `wireshark`/`tshark`.

You are welcome to create additional `make` rules in the Makefile, but we ask that you keep the existing `final` and `checkpoint`
rules, as we will use them for lab grading.


### Generating documentation

We want you to get in the habit of documenting your code in a way that leads to detailed, easy-to-read/-navigate package 
documentation for your game server package. Our Makefile includes a `docs` rule that will pipe the output of the `go doc` command
into a text file that is reasonably easy to read.  We will use this output for the manually graded parts of the lab, so good
comments are valuable.


### Questions?

If there is any part of the initial repository, environment setup, lab requirements, or anything else, please do not hesitate
to ask.  We're here to help!

Player must wait for a response after issuing a command

### Credit 

This project is the work of the following individuals:
Name: Yifan Lin, Jiawei Zhu
AndrewID: yifanl2, jiaweizh

### Documentation

The additional file containing discussions of the overall framework and failure scenarios of this project is contained in the Description.md file. Please let us know if there are possible issues related to this.

### Requests/Responses:
1. join game: {"cmd": "JOIN", "name": <player name>} -> {"status": ["success"|"fail"], "state": ["WAITING"|"FULL"|"READY"], "leader": <leader's name>}
2. start game: {"cmd": "START", "name": <player name>} -> {"status": ["success"|"fail"], "reason": ["already started"|"not a leader"|"not enough players"], "wait": "<number of people to wait>", "leader": <leader's name>}
3. reconnect a game: {"cmd": "RECONN", "name": <player name>} -> {"status": ["success"|"fail"], "leader": <leader's name>, "state": ["WAITING"|"FULL"|"READY"]}
4. upload a file: {"cmd": "UPLOAD", "name": <player name>, "filename": <file name>} -> {"status": ["success"|"fail"], "path": <path to store the file>} -> {"status": ["success"|"fail"]}
5. a player disconnects: {"cmd": "DISCONN", "name": <player name>} -> nothing
6. picker uploads a word: {"cmd": "RANDOM_WORD", "name": <player name>, "word": <word>} -> {"status": ["success"|"fail"], "reason": ["not a picker"|"not a valid choice"|"file not ready"], "picker": <picker's name>}
7. player sends its guess to the game: {"cmd": "WORD_COUNT", "name": <player name>, "guess": <this player's guess>} -> {"status": ["success"|"fail"], "reason": ["did not join the game"|"not ready for guesses"|"invalid format"]}
8. player sends restart: {"cmd": "RESTART", "name": <player name>} -> {"status": ["success"|"fail"]}
9. player sends close: {"cmd": "CLOSE", "name": <player name>} -> {"status": ["success"|"fail"]}
10. player says goodbye: {"cmd": "GOODBYE", "name": <player name>} -> {"status": "success"}

### Notifications:
1. notify the leader when the game is ready to start: {"gameID": <this game's id>, "msg": "READY"}
2. notify non-leaders that the game started: {"gameID": <this game's id>, "msg": "STARTED", "leader": <leader's name>}
3. notify non-pickers when a file is uploaded: {"gameID": <this game's id>, "msg": "UPLOADED"}
4. notify the pickers when a file is uploaded: {"gameID": <this game's id>, "msg": "PICK", "filename": <file name>}
5. notify everyone of the new leader: {"gameID": <this game's id>, "msg": "NEW_LEADER", "leader": <leader's name>}
6. notify everyone about the selected word: {"gameID": <this game's id>, "msg": "WORD_SELECTED", "word": <word>}
7. notify everyone of the winner: {"gameID": <this game's id>, "msg": "WINNER", "name": <winner's name>}
8. notify everyone that the game has restarted: {"gameID": <this game's id>, "msg": "RESTARTED"}
9. notify everyone that the game has closed: {"gameID": <this game's id>, "msg": "CLOSED"}
10. notify everyone to gracefully exit: {"gameID": <this game's id>, "msg": "EXIT"}