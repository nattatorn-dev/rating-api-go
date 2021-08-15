package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Service struct {
	redis *redisClient
}

func (s *Service) Handler(fn func(*Service, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(s, w, r)
	}
}

type RegisterRatingLeaderboardRequest struct {
	UserId    string `json:"userId"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

type RegisterRatingLeaderboardResponse struct {
	UserId    string `json:"userId"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

type Error struct {
	Error        int    `json:"error"`
	ErrorMessage string `json:"message,omitempty"`
}

func (service *Service) Register(w http.ResponseWriter, r *http.Request) {
	var newUser RegisterRatingLeaderboardRequest

	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	registering := map[string]interface{}{
		"firstname": newUser.FirstName,
		"lastname":  newUser.LastName,
	}

	err = service.redis.setHMSet(fmt.Sprintf("users:%s", newUser.UserId), registering)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (service *Service) GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userId := params["userId"]

	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	res, err := service.redis.getHMKey(fmt.Sprintf("users:%s", userId))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(res)
}

type RateUserRequest struct {
	UserId string  `json:"userId"`
	Score  float64 `json:"score"`
}

func MonthRatingSummary() string {
	currentTime := time.Now()
	year, month, _ := currentTime.Date()
	monthAndYear := fmt.Sprintf("%02d-%d", month, year)

	return monthAndYear
}

func userRatingAccumulateLeaderboardMonthKey() string {
	return fmt.Sprintf("leaderboard.user.rating.accumulate:%s", MonthRatingSummary())
}

func userRatingLeaderboardMonthKey() string {
	return fmt.Sprintf("leaderboard.user.rating:%s", MonthRatingSummary())
}

func rateCountMonthKey(userId string) string {
	return fmt.Sprintf("count.user.rated:%s:%s", MonthRatingSummary(), userId)
}

type RateUserResponse struct {
	Score float64 `json:"score"`
}

func (service *Service) RateUser(w http.ResponseWriter, r *http.Request) {
	var rating RateUserRequest

	err := json.NewDecoder(r.Body).Decode(&rating)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	rateCountMonthKey := rateCountMonthKey(rating.UserId)
	totalRated, err := service.redis.setIncrement(rateCountMonthKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	ratingScoreMonthKey := userRatingAccumulateLeaderboardMonthKey()
	score := rating.Score
	member := rating.UserId

	totalScore, err := service.redis.setSortIncrementBy(ratingScoreMonthKey, score, member)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	userRatingLeaderboardMonthKey := userRatingLeaderboardMonthKey()
	avgScore := totalScore / float64(totalRated)

	_, err = service.redis.setSortKey(userRatingLeaderboardMonthKey, avgScore, member)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (service *Service) GetUserRatingById(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userId := params["userId"]

	if userId == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	key := userRatingLeaderboardMonthKey()
	score, err := service.redis.getSortScore(key, userId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	userScoreResponse := RateUserResponse{Score: score}
	json.NewEncoder(w).Encode(userScoreResponse)
}

type UserRatingLeaderboardResponse struct {
	UserId string  `json:"userId"`
	Score  float64 `json:"score"`
}

func (service *Service) UserRatingLeaderboard(w http.ResponseWriter, r *http.Request) {
	max := r.URL.Query().Get("max")

	var maxNumber int64
	maxNumber = 5
	var err error

	if max != "" {
		maxNumber, err = strconv.ParseInt(max, 0, 64)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(Error{
				http.StatusInternalServerError,
				"max should be a number",
			})
			return
		}
	}

	key := userRatingLeaderboardMonthKey()
	ranks, err := service.redis.getRank(key, maxNumber)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Error{
			http.StatusInternalServerError,
			err.Error(),
		})
		return
	}

	leaderBoard := []UserRatingLeaderboardResponse{}
	for _, rank := range ranks {
		rankModel := UserRatingLeaderboardResponse{
			UserId: rank.Member,
			Score:  rank.Score,
		}
		leaderBoard = append(leaderBoard, rankModel)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(leaderBoard)
}

type RedisHealthCheck struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type HealthCheck struct {
	Redis  RedisHealthCheck `json:"redis"`
	Status string           `json:"status"`
}

func (service *Service) Healthz(w http.ResponseWriter, r *http.Request) {
	err := service.redis.HealthCheck()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		serviceUnavailable := HealthCheck{
			Redis:  RedisHealthCheck{Status: "DOWN", Error: err.Error()},
			Status: "DOWN",
		}

		json.NewEncoder(w).Encode(serviceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	ok := HealthCheck{
		Redis:  RedisHealthCheck{Status: "UP"},
		Status: "UP",
	}

	json.NewEncoder(w).Encode(ok)
}
