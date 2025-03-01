package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var a *ElectionApp

func TestMain(m *testing.M) {
	a = NewElectionApp()
	code := m.Run()
	os.Exit(code)
}

func TestNoResults(t *testing.T) {
	_ = a.results.Reset()
	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}
	// Seats per party LAB = 0, CON = 0, LD = 0
	// Winner = none
	if scoreboard == nil {
		t.Errorf("Scoreboard was nil")
		return
	}
}

func Test5Results(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(5)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}
	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}
	// assert LD == 1
	// assert LAB = 4
	// assert winner = noone
	// Winner = none
	if scoreboard == nil {
		t.Errorf("Scoreboard was nil")
		return
	}
}

func Test100Results(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(100)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}
	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}
	// assert LD == 12
	// assert LAB == 56
	// assert CON == 31
	// Winner = none
	if scoreboard == nil {
		t.Errorf("Scoreboard was nil")
		return
	}
}

func Test554Results(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(554)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}
	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}
	// assert LD == 52
	// assert LAB = 325
	// assert CON = 167
	// assert winner = LAB
	if scoreboard == nil {
		t.Errorf("Scoreboard was nil")
		return
	}
}

func TestAllResults(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(650)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}
	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}
	// assert LD == 62
	// assert LAB == 349
	// assert CON == 210
	// assert sum = 650
	if scoreboard == nil {
		t.Errorf("Scoreboard was nil")
		return
	}
}

func TestScoreboardCalculation(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(650)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}

	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}

	r, _ := json.Marshal(scoreboard)
	log.Println("Scoreboard: ", string(r))

	expectedSeats := map[string]int{
		"LD":  62,
		"LAB": 349,
		"CON": 210,
	}
	for party, expected := range expectedSeats {
		if scoreboard.Seats[party] != expected {
			t.Errorf("Expected %s to have %d seats, but got %d", party, expected, scoreboard.Seats[party])
		}
	}

	if scoreboard.Winner != "LAB" {
		t.Errorf("Expected LAB to be the winner, but got %s", scoreboard.Winner)
	}
}

func TestVoteShareCalculation(t *testing.T) {
	_ = a.results.Reset()
	err := postResults(650)
	if err != nil {
		t.Errorf("Error posting results: %v", err)
		return
	}

	scoreboard, err := getScoreboard(t)
	if err != nil {
		t.Errorf("Error getting scoreboard: %v", err)
		return
	}

	expectedVoteShare := map[string]float64{
		"LD":  22.00,
		"LAB": 35.00,
		"CON": 32.00,
	}
	for party, expected := range expectedVoteShare {
		if math.Round(scoreboard.VoteShare[party]) != math.Round(expected) {
			t.Errorf("Expected %s to have %.0f%% of the votes, but got %.0f%%",
				party, expected, scoreboard.VoteShare[party])
		}
	}
}

func postResults(results int) error {
	for i := 1; i <= results; i++ {
		filename := fmt.Sprintf("sample-election-results/result%03d.json", i)
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		req, _ := http.NewRequest("POST", "/result", bytes.NewReader(data))
		rr := httptest.NewRecorder()
		a.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			return errors.New(fmt.Sprintf("wrong status code, expected 201 but got: %d", rr.Code))
		}
	}
	return nil
}

func getScoreboard(t *testing.T) (*Scoreboard, error) {
	req, _ := http.NewRequest("GET", "/scoreboard", nil)
	response := executeRequest(req)

	if response.Code != http.StatusOK {
		t.Errorf("Expected response code %d. Got %d\n", http.StatusOK, response.Code)
		return nil, errors.New("wrong status code")
	}
	scoreboard, err := marshalResponseToScoreboard(response)
	if err != nil {
		return nil, err
	}
	return &scoreboard, nil
}

func marshalResponseToScoreboard(response *httptest.ResponseRecorder) (Scoreboard, error) {
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Scoreboard{}, nil
	}
	var scoreboard Scoreboard
	err = json.Unmarshal(b, &scoreboard)
	if err != nil {
		return Scoreboard{}, nil
	}
	return scoreboard, nil
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.router.ServeHTTP(rr, req)
	return rr
}
