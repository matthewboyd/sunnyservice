package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool" //for sql
	pb "github.com/matthewboyd/sunnyservice/pb"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

type Activities struct {
	Name     string
	Postcode string
	Sunny    bool
}

type Handler struct {
	Logger log.Logger
	Db     pgxpool.Pool
	Redis  redis.Client
}

type Weather struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
		SeaLevel  int     `json:"sea_level"`
		GrndLevel int     `json:"grnd_level"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int `json:"dt"`
	Sys struct {
		Type    int    `json:"type"`
		ID      int    `json:"id"`
		Country string `json:"country"`
		Sunrise int    `json:"sunrise"`
		Sunset  int    `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

type server struct {
	pb.UnimplementedSunnyServiceServer
	Handler
}

func (h *server) GetSunnyActivities(ctx context.Context, in *pb.GetSunnyActivitiesParams) (*pb.Activity, error) {
	var activityList []Activities
	var a Activities

	rows, err := h.Db.Query(ctx, "SELECT * FROM activities where sunny = $1", true)
	if err != nil {
		log.Fatalln("an error occurred in the sunny query", err)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&a.Name, &a.Postcode, &a.Sunny)
		if err != nil {
			log.Fatalln("Error when scanning the db rows", err)
		}
		activityList = append(activityList, a)
	}
	log.Println("activityList", activityList)
	if err != nil {
		log.Fatalln("An error occurred", err)
	}
	var discardedActivityList []Activities
	choosenActivity, _ := h.retrieveActivity(ctx, activityList, discardedActivityList, true, 0)
	return &pb.Activity{choosenActivity.Name, choosenActivity.Postcode}, nil
}

func (h *server) retrieveActivity(ctx context.Context, newActivityList []Activities, discardedActivityList []Activities, sunny bool, tries int) (Activities, error) {
	if tries > 3 {
		return Activities{}, errors.New("we're having difficulties finding a sunny activity, why not try an allWeather activity")
	}
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	randomNumber := r1.Intn(len(newActivityList))
	choosenActivity := newActivityList[randomNumber]
	if sunny {
		//check cache
		value, err := h.Redis.Get(ctx, choosenActivity.Postcode).Result()
		if err == redis.Nil {
			// we want to call the API
			weather := choosenActivity.GetWeather()

			_ = h.Redis.Set(ctx, choosenActivity.Postcode, weather, time.Minute*10).Err()

			if weather != "Rain" && weather != "Snow" && weather != "Drizzle" {
				return choosenActivity, nil
			} else {
				discardedActivityList = append(discardedActivityList, choosenActivity)
				newActivityList = h.RemoveIndex(newActivityList, randomNumber)
				tries++
				return h.retrieveActivity(ctx, newActivityList, discardedActivityList, true, tries)
			}
		} else if err != nil {
			return Activities{}, err
		} else {
			// build response
			if value != "Rain" && value != "Snow" && value != "Drizzle" {
				return choosenActivity, nil
			} else {
				discardedActivityList = append(discardedActivityList, choosenActivity)
				newActivityList = h.RemoveIndex(newActivityList, randomNumber)
				tries++
				return h.retrieveActivity(ctx, newActivityList, discardedActivityList, true, tries)
			}
		}

	} else {
		return choosenActivity, nil
	}
}

func (a *Activities) GetWeather() string {

	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?appid=%s&q=%s", os.Getenv("WEATHER_API_KEY"), a.Postcode)
	response, err := http.Get(url)
	if err != nil {
		log.Fatalln("retrieving the weather", err)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalln("retriving the body", err)
	}
	var weather Weather

	if err := json.Unmarshal(body, &weather); err != nil {
		log.Fatalln("error unmarshalling response to json", err)
	}
	return weather.Weather[0].Main
}

func (h *server) RemoveIndex(s []Activities, index int) []Activities {
	return append(s[:index], s[index+1:]...)
}

func main() {
	lis, err := net.Listen("tcp", ":6666")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterSunnyServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
