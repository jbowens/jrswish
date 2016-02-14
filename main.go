package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/jbowens/nbagame"
	"github.com/jbowens/nbagame/data"
)

const (
	cavsTeamID      = 1610612739
	jrSmithPlayerID = 2747
)

var (
	twitter *anaconda.TwitterApi
)

func init() {
	anaconda.SetConsumerKey(os.Getenv("CONSUMER_KEY"))
	anaconda.SetConsumerSecret(os.Getenv("CONSUMER_SECRET"))
	twitter = anaconda.NewTwitterApi(
		os.Getenv("ACCESS_TOKEN"),
		os.Getenv("ACCESS_TOKEN_SECRET"),
	)
	rand.Seed(time.Now().UnixNano())
}

type shot struct {
	GameID    string
	WallClock string
}

func tweetStatus(game *data.Game, event *data.Event) (string, bool) {
	attrs := map[data.ShotAttemptAttribute]bool{}
	for _, attr := range event.ShotAttributes {
		attrs[attr] = true
	}

	switch {
	case attrs[data.Missed]:
		return "", false
	case event.PeriodTime == "0:00", event.PeriodTime == "0:01",
		event.PeriodTime == "0:02", event.PeriodTime == "0:03":
		return "THE BUZZER BEATER SWWWIISSHSHHHHH @TheRealJRSmith #TeamSwish", true
	case attrs[data.AlleyOop]:
		return "@TheRealJRSmith allleeeyyy ooooooop!!!! #TeamSwish", true
	case attrs[data.ThreePointer] && attrs[data.StepBack]:
		return "That step back SWISH! #TeamSwish", true
	case attrs[data.ThreePointer] && attrs[data.Fadeaway]:
		return "JR fadingggggg..... SWISH!", true
	case attrs[data.ThreePointer] && rand.Intn(1000) == 5:
		return "Now watch him swish... swish... watch him nae nae @TheRealJRSmith #TeamSwish", true
	case attrs[data.ThreePointer]:
		return "Swish!", true
	case attrs[data.Dunk]:
		return "JR throwing it DOWN! #TeamSwish", true
	}

	return "", false
}

func retrieveCavsGame(t time.Time) (*data.Game, error) {
	gamesToday, err := nbagame.API.Games.ByDate(t)
	if err != nil {
		return nil, err
	}

	var game *data.Game
	for _, g := range gamesToday {
		if g.HomeTeamID == cavsTeamID || g.VisitorTeamID == cavsTeamID {
			game = g
		}
	}
	if game.Status == data.Live {
		return game, nil
	}
	return nil, nil
}

func main() {
	var mostRecentGame *data.Game
	eventsProcessed := map[shot]*data.Event{}

	for t := range time.Tick(time.Minute) {
		t = t.Local()
		if t.Month() >= time.July && t.Month() <= time.October {
			// Ignore because it's not during the season.
			continue
		}
		if t.Hour() >= 1 && t.Hour() < 12 {
			// Ignore because no games are played between these hours.
			continue
		}

		// Update the game to watch every 10 minutes.
		if t.Minute()%5 == 0 {
			g, err := retrieveCavsGame(t)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error retrieving Cavs game: %s\n", err)
				continue
			}
			if g != nil && mostRecentGame == nil {
				fmt.Printf("Looks like game %#v just started...\n", g)
			}
			mostRecentGame = g
		}
		if mostRecentGame == nil {
			continue
		}

		events, err := nbagame.API.Games.PlayByPlay(mostRecentGame.ID.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error retrieving play-by-play: %s\n", err)
			continue
		}

		for _, e := range events {
			if !e.Is(data.ShotAttempt) {
				continue
			}
			if len(e.InvolvedPlayers) == 0 {
				continue
			}
			if e.InvolvedPlayers[0].ID != jrSmithPlayerID {
				continue
			}

			shotID := shot{
				GameID:    e.GameID.String(),
				WallClock: e.WallClock,
			}
			if _, ok := eventsProcessed[shotID]; ok {
				continue
			}
			eventsProcessed[shotID] = e

			// Tweet that shit.
			status, ok := tweetStatus(mostRecentGame, e)
			if !ok {
				continue
			}

			_, err := twitter.PostTweet(status, url.Values{})
			if err != nil {
				fmt.Fprintf(os.Stderr, "error tweeting shot %#v: %s\n", e, err)
				continue
			}
		}
	}
}
