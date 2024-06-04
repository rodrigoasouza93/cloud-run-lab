package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rodrigoasouza93/cloud-run-lab/configs"
	"github.com/rodrigoasouza93/cloud-run-lab/internal/dto"
	"github.com/rodrigoasouza93/cloud-run-lab/internal/vo"
)

func main() {
	http.HandleFunc("GET /{cep}", getWeatherHandler)
	http.ListenAndServe(":8080", nil)
}

func getWeatherHandler(w http.ResponseWriter, r *http.Request) {
	rawCep := r.PathValue("cep")
	cep, err := vo.NewCep(rawCep)
	if err != nil {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}
	locationURL := "https://viacep.com.br/ws/" + cep.Value() + "/json"
	fmt.Println(locationURL)
	respLocation, err := http.Get(locationURL)
	if err != nil || respLocation.StatusCode != http.StatusOK {
		fmt.Println(respLocation)
		fmt.Println(err)
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}
	defer respLocation.Body.Close()

	var decodedLocation dto.LocationResponse
	err = json.NewDecoder(respLocation.Body).Decode(&decodedLocation)
	if err != nil {
		http.Error(w, "error decoding location: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if decodedLocation.Error {
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	config := configs.LoadConfig(".")
	weatherAPIURL := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", config.WeatherAPIKey, url.QueryEscape(decodedLocation.Locale))
	respWeather, err := http.Get(weatherAPIURL)
	if respWeather.StatusCode != http.StatusOK || err != nil {
		fmt.Println(weatherAPIURL)

		http.Error(w, "can not get weather", respWeather.StatusCode)
		return
	}
	defer respWeather.Body.Close()

	var decodedWeather dto.WeatherResponse
	if err := json.NewDecoder(respWeather.Body).Decode(&decodedWeather); err != nil {
		http.Error(w, "error decoding weather: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response := dto.WeatherOutput{
		Temp_C: decodedWeather.Current.TempC,
		Temp_F: decodedWeather.Current.TempF,
		Temp_K: getKelvinTemp(decodedWeather.Current.TempC),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getKelvinTemp(celsius float64) float64 {
	return celsius + 273.15
}
