# Overview
### Getting started

To get started with this project, first you need to install Go:
```
sudo apt update; sudo apt -y upgrade
sudo apt install -y git make

# Install Go
sudo snap install go --classic
go env -w GO111MODULE=auto
```
to start working on the lab. Everything else is controlled by `make` using the `go` command and shell scripts. For
more details about installing `go`, see [https://go.dev/doc/install](https://go.dev/doc/install).


### Repository contents

```
\---Distributed_Multiclient_Server
        +---src
        |   \---gameServer
        |   |   +---game.go
        |   |   +---gameServer.go
        |   |   +---gameServer_test.go
        |   |   +---messages.go
        |   |   +---player.go
        |   |   \---test.txt
        |   \---go.mod
        +---Makefile
        \---README.md
```

### Testing the game server

For example, to run a game server on port `14736` and connect multiple players, execute
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
terminal.

## Credits

This project is the work of the following individuals:<br>
Name: Allen Zhu, Ethan Lin, Patrick Samantis
<br><br>
# Framework of Server-Client Interaction
## Server's Core Functions:
- Connection Establishment: Acts as the nerve center, managing TCP connections for stable, ordered, and error-checked communication.
- Request Processing: Interprets client commands, maintains game logic and synchronizes state across various games and participants.
- State Preservation: Monitors and updates the status of games and players, adeptly handling anomalies such as client disconnections.

## Client's Engagement:
- Command Transmission: Clients send structured commands to interact with the game, influencing the game's flow and their roles.
- Update Reception: Actively listens and reacts to server's instructions, adapting to evolving game scenarios.

## Data Exchange
- Communication Protocols: Uses TCP/IP for robust data transmission, ensuring reliability and consistency in the multiplayer environment.
- Structured Data Exchange: Employs JSON format for exchanging data, offering a universally understood protocol for both server and client sides to decode messages efficiently.

## Detailed Game Mechanics and Handling of Requests/Responses
- Joining a Game (JOIN):
    - Initiation: Players express interest in participating by sending a JOIN request.
    - Server's Logic: Examines the availability of games, considering factors like current state (WAITING, FULL, READY) and player capacity.
    - Outcome: Responses indicate successful inclusion or reasons for rejection, setting the foundation for ensuing gameplay.

- Game Commencement (START):
    - Leader's Prerogative: The designated leader initiates the game start.
    - Validation Process: The server ensures the requestor's leadership status and checks for sufficient player count.
    - Transitioning States: Affirmative responses shift the game into action, while negatives detail impediments like insufficient players or incorrect leadership.

- Reconnecting (RECONN):
    - Facilitating Continuity: Designed to reintegrate players post-disconnection.
    - State Update: The server reassesses and informs players of their current game status, ensuring seamless re-entry into the gameplay.

- File Upload Mechanism (UPLOAD):
    - Pivotal for Gameplay: Leaders upload textual content, pivotal for the guessing game.
    - Verification and Storage: The server confirms the uploader's role and checks for duplicate filenames.
    - Feedback Loop: Responses reflect successful uploads or highlight conflicts, triggering subsequent gameplay phases.

- Handling Disconnections (DISCONN):
    - Graceful Management: Addresses players' unexpected departure.
    - Server's Adaptability: Updates game states accordingly, managing role reassignments to preserve game integrity.

- Word Selection (RANDOM_WORD):
    - Picker's Privilege: Designated players select words from the uploaded content.
    - Word Validation: The server checks the picker's role and the word's presence and uniqueness in the text.
    - Informative Responses: Players are notified of the success or detailed reasons for rejection.

- Guessing Word Counts (WORD_COUNT):
    - Player Involvement: Participants submit their estimates of the word count.
    - Readiness and Accuracy Checks: The server confirms the player's game involvement and readiness for guessing, validating the guess format.
    - Response Dynamics: Success or failure responses are provided, with reasons for any rejections specified.

- Game Restart (RESTART):
    - Leadership Command: Enables the game leader to initiate a new round.
    - Server's Reconfiguration: Resets the game while retaining player composition.
    - Renewed Notifications: Alerts players of the game's recommencement.

- Game Closure (CLOSE):
    - Concluding Authority: Leaders can end the game, dissolving its structure.
    - State Dissolution: Server processes closure requests, clearing related game data.
    - Participant Notification: Players are informed about the game's termination.

- Player's Departure (GOODBYE):
    - Exit Strategy: Players use this to gracefully exit the server's realm.
    - Leadership Games Closure: The server concludes games under the departing player's lead.
    - Role Reassignments: Adjusts roles in ongoing games to maintain continuity.

## Server Notifications and Real-Time Updates
- Dynamic Server Alerts: Key to keeping players informed about game progress, role assignments, and changes.
- Variety of Notifications: Ranging from game readiness (READY) to winner announcements (WINNER), these alerts are pivotal in guiding player actions and understanding of the game's current state.

## Backend Infrastructure
- Server Infrastructure:
    - Robust Listener Mechanism: Continuously listens for incoming connections, channeling them into distinct client routines.
    - Game and Player Management: Orchestrates game sessions, tracks player statuses, and manages their reconnections.
    - Concurrent Processing: Leverages Go's goroutines and channels for concurrent handling of client requests and game progressions.
    - Storage and Recovery: Implements storage mechanisms for game states, enabling resumption and recovery of games.

- Client Handling:
    - Command Interpretation and Sending: Clients communicate their actions through defined commands, each triggering specific server-side reactions.
    - Reactive to Server Updates: Actively adjusts to real-time game changes, reflecting server-sent updates in the gameplay.

- Error Handling and Security:
    - Input Validation and Error Responses: Ensures inputs meet expected formats and rules, providing clarity in error messages.
    - Security Measures: Protects against malicious inputs and maintains game integrity by aligning actions with player roles and game rules.

## Features and Scalability
- Scalable Architecture: Designed to handle multiple games and players simultaneously, showcasing the capability to scale as per demand.
- Advanced Game Mechanics: Incorporates features like word selection, file uploads, and guessing mechanisms, making the gameplay engaging and complex.

## Failure scenarios
- Client Disconnections
    - Scenario:
        - Players might unexpectedly disconnect due to network issues, client crashes, or intentional exit.
        - Unhandled disconnections can leave the game in an inconsistent state, especially if the disconnected player was in a critical role (e.g., leader or picker).
    - Impact:
        - If the leader disconnects, the game may become unmanageable or stuck.
        - If a picker disconnects after a file upload but before word selection, the game might stall.

- Server Overload
    - Scenario:
        - The server might become overloaded due to a high number of concurrent games or players, especially if each game involves complex operations and real-time interactions.
    - Impact:
        - Server performance might degrade, leading to delays in processing commands, timeouts, or even server crashes.
        - Players might experience lag or unresponsiveness, impacting the game experience.
- Synchronization Issues
    - Scenario:
        - Race conditions might arise when multiple goroutines access shared resources like player states or game channels without proper synchronization.
        - Deadlocks can occur if concurrent processes wait on each other indefinitely.
    - Impact:
        - Race conditions can lead to inconsistent game states or errors like writing to closed channels.
        - Deadlocks can freeze the game, requiring server restarts to resolve.
- Data Persistence and Recovery Issues
    - Scenario:
        - The server might crash or be intentionally restarted for maintenance or updates.
        - In the absence of persistent storage, ongoing games and player progress might be lost.
    - Impact:
        - Players might lose their game progress, leading to frustration and decreased trust in the system.
        - Repeated crashes without recovery options can diminish the game's popularity and player base.
- Security Vulnerabilities
    - Scenario:
        - Network-based applications are susceptible to various security threats, such as Denial of Service (DoS) attacks, unauthorized access, or data tampering.
    - Impact:
        - Server availability can be compromised, disrupting service for legitimate players.
        - Sensitive player data, if not adequately protected, can be exposed or manipulated.
- Faulty Game Logic or Command Handling
    - Scenario:
        - Bugs in the game logic or improper handling of client commands can lead to unexpected behavior.
    - Impact:
        - Players might exploit bugs for unfair advantages.
        - The game might enter invalid states, impacting player experience and trust.
- Resource Leaks
    - Scenario:
        - Inadequate resource management, such as not releasing file handles or not closing network connections, can lead to resource leaks.
    - Impact:
        - Over time, resource leaks can degrade server performance and stability.

## Documentation

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
