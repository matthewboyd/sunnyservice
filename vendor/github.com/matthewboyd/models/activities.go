package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Activities struct {
	Name     string
	Postcode string
	Sunny    bool
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
