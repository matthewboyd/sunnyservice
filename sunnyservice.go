package main

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool" //for sql
	"github.com/matthewboyd/models"
	pb "github.com/matthewboyd/sunnyservice/pb"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"net"
	"time"
)

type Handler struct {
	Logger log.Logger
	Db     pgxpool.Pool
	Redis  redis.Client
}

type server struct {
	pb.UnimplementedSunnyServiceServer
	Handler
}

type Activities models.Activities

func (h *server) GetSunnyActivities(ctx context.Context, in *pb.GetSunnyActivitiesParams) (*pb.Activity, error) {
	var activityList []models.Activities
	var a models.Activities

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
	var discardedActivityList []models.Activities
	choosenActivity, _ := h.retrieveActivity(ctx, activityList, discardedActivityList, true, 0)
	return &pb.Activity{choosenActivity.Name, choosenActivity.Postcode}, nil
}

func (h *server) retrieveActivity(ctx context.Context, newActivityList []models.Activities, discardedActivityList []models.Activities, sunny bool, tries int) (models.Activities, error) {
	if tries > 3 {
		return models.Activities{}, errors.New("we're having difficulties finding a sunny activity, why not try an allWeather activity")
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
			return models.Activities{}, err
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

func (h *server) RemoveIndex(s []models.Activities, index int) []models.Activities {
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
