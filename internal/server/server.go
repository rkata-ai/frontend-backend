package server

import (
	"encoding/json"
	"log"
	"net/http"

	"frontend-backend/internal/storage"

	"github.com/gorilla/mux"
)

// Server представляет HTTP-сервер
type Server struct {
	store  *storage.PostgresStorage
	router *mux.Router
}

// NewServer создает новый экземпляр Server
func NewServer(store *storage.PostgresStorage) *Server {
	s := &Server{
		store:  store,
		router: mux.NewRouter(),
	}
	s.setupMiddleware()
	s.routes()
	return s
}

// setupMiddleware настраивает middleware для сервера
func (s *Server) setupMiddleware() {
	s.router.Use(corsMiddleware)
}

// routes инициализирует маршруты сервера
func (s *Server) routes() {
	s.router.HandleFunc("/stocks", s.getStocksHandler).Methods("GET")
	s.router.HandleFunc("/predictions/{ticker}", s.getPredictionsByTickerHandler).Methods("GET")
	s.router.HandleFunc("/stocks/{ticker}/history", s.getStockHistoryHandler).Methods("GET")
}

// ServeHTTP реализует интерфейс http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// getStocksHandler обрабатывает запрос на получение списка акций
func (s *Server) getStocksHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("GET /stocks - получение списка акций")
	w.Header().Set("Content-Type", "application/json")

	stocks, err := s.store.GetStocks()
	if err != nil {
		log.Printf("Ошибка при получении акций: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Возвращаем %d акций", len(stocks))
	json.NewEncoder(w).Encode(stocks)
}

// getPredictionsByTickerHandler обрабатывает запрос на получение прогнозов по тикеру
func (s *Server) getPredictionsByTickerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	ticker := params["ticker"]

	log.Printf("GET /predictions/%s - получение прогнозов для тикера: '%s'", ticker, ticker)

	predictions, err := s.store.GetPredictionsByTicker(ticker)
	if err != nil {
		log.Printf("Ошибка при получении прогнозов для тикера '%s': %v", ticker, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Найдено %d прогнозов для тикера '%s'", len(predictions), ticker)
	json.NewEncoder(w).Encode(predictions)
}

// corsMiddleware добавляет CORS заголовки
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем запросы с localhost:5173 (Vite dev server)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Обрабатываем preflight запросы
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getStockHistoryHandler обрабатывает запрос на получение истории цен акции
func (s *Server) getStockHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	ticker := params["ticker"]

	log.Printf("GET /stocks/%s/history - получение истории цен для тикера: '%s'", ticker, ticker)

	history, err := s.store.GetStockPriceHistory(ticker)
	if err != nil {
		log.Printf("Ошибка при получении истории цен для тикера '%s': %v", ticker, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Найдено %d записей истории цен для тикера '%s'", len(history), ticker)
	json.NewEncoder(w).Encode(history)
}
