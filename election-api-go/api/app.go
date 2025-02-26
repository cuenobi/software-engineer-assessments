package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ElectionApp struct {
	router  *mux.Router
	results ResultService
}

func NewElectionApp() *ElectionApp {
	router := mux.NewRouter()
	app := ElectionApp{router, NewResultService()}
	router.HandleFunc("/result", app.newResult).Methods("POST")
	router.HandleFunc("/result/{id:[0-9]+}", app.getResult).Methods("GET")
	router.HandleFunc("/scoreboard", app.getScoreboard).Methods("GET")
	return &app
}

func (a *ElectionApp) Run(port int) {
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), a.router))
}

type ConstituencyResult struct {
	Id           int           `json:"id"`
	Name         string        `json:"name"`
	SeqNo        int           `json:"seqNo"`
	PartyResults []PartyResult `json:"partyResults"`
}

type PartyResult struct {
	Party string      `json:"party"`
	Votes uint        `json:"votes"`
	Share json.Number `json:"share"`
}

type Scoreboard struct {
	Seats      map[string]int     `json:"seats"`
	Winner     string             `json:"winner"`
	TotalVotes map[string]uint    `json:"total_votes"`
	VoteShare  map[string]float64 `json:"vote_share"`
}

func (a ElectionApp) newResult(writer http.ResponseWriter, request *http.Request) {
	var cr ConstituencyResult
	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()
	err := decoder.Decode(&cr)
	if err != nil {
		a.writeResponse(writer, 500, err.Error())
		return
	}
	err = a.results.AddResult(cr)
	if err != nil {
		a.writeResponse(writer, 500, err.Error())
		return
	}
	a.writeResponse(writer, 201, "Created")
}

func (a ElectionApp) getResult(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		a.writeResponse(writer, 400, fmt.Sprintf("result: %s is not an int", vars["id"]))
		return
	}
	result, err := a.results.GetResult(id)
	if err != nil {
		_, ok := err.(NotFound)
		if ok {
			a.writeResponse(writer, 404, err.Error())
		} else {
			a.writeResponse(writer, 500, err.Error())
		}
		return
	}
	status := 200
	response, _ := json.Marshal(result)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	writer.Write(response)
}

// getScoreboard handles HTTP requests to fetch the election scoreboard.
// It calculates and returns the number of seats, total votes, and vote share for each party,
// and determines the winning party if any.
func (a ElectionApp) getScoreboard(writer http.ResponseWriter, request *http.Request) {
	// Retrieve all election results from the result service
	results, err := a.results.GetAll()
	if err != nil {
		a.writeResponse(writer, 500, err.Error())
		return
	}

	// Initialize maps to store calculated seats, total votes, and vote share for each party
	seats := make(map[string]int)
	totalVotes := make(map[string]uint)
	voteShare := make(map[string]float64)
	var totalAllVotes uint

	// Calculate seats and total votes for each party
	for _, result := range results {
		var winnerParty string
		var maxVotes uint
		for _, partyResult := range result.PartyResults {
			totalVotes[partyResult.Party] += partyResult.Votes
			totalAllVotes += partyResult.Votes
			if partyResult.Votes > maxVotes {
				maxVotes = partyResult.Votes
				winnerParty = partyResult.Party
			}
		}
		// Increment seat count for the party with the most votes in the constituency
		if winnerParty != "" {
			seats[winnerParty]++
		}
	}

	// Determine the winning party if it has reached the required seat count
	var winner string
	for party, seat := range seats {
		if seat >= 325 {
			winner = party
			break
		}
	}

	// Calculate the percentage share of votes for each party
	for party, votes := range totalVotes {
		voteShare[party] = (float64(votes) / float64(totalAllVotes)) * 100
	}

	// Construct the scoreboard response with calculated data
	result := Scoreboard{
		Seats:      seats,
		Winner:     winner,
		TotalVotes: totalVotes,
		VoteShare:  voteShare,
	}

	// Marshal the result into JSON and write it to the response
	response, _ := json.Marshal(result)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	writer.Write(response)
}

func (a ElectionApp) writeResponse(writer http.ResponseWriter, status int, body string) {
	writer.WriteHeader(status)
	writer.Write([]byte(body))
}
