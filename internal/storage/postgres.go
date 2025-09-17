package storage

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Stock представляет акцию из таблицы stocks
type Stock struct {
	ID     int64  `json:"id"`
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}

// Prediction представляет прогноз, как описано для фронтенда
type Prediction struct {
	ID                  int64    `json:"ID"`
	MessageID           int64    `json:"MessageID"`
	StockID             int64    `json:"StockID"`
	PredictionType      *string  `json:"PredictionType"`
	TargetPrice         *float64 `json:"TargetPrice"`
	TargetChangePercent *float64 `json:"TargetChangePercent"`
	Period              *string  `json:"Period"`
	Recommendation      *string  `json:"Recommendation"`
	Direction           *string  `json:"Direction"`
	JustificationText   *string  `json:"JustificationText"`
	Message             *string  `json:"Message"`     // Полный текст сообщения из таблицы messages
	PredictedAt         string   `json:"PredictedAt"` // ISO-формат даты или Unix timestamp
}

// StockPriceHistory представляет историческую цену акции
type StockPriceHistory struct {
	StockID   int64   `json:"StockID"`
	Timestamp string  `json:"Timestamp"`
	Price     float64 `json:"Price"`
	Volume    int64   `json:"Volume,omitempty"`
}

// PostgresStorage реализует хранилище данных для PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage создает новый экземпляр PostgresStorage
func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// GetStocks извлекает список акций из базы данных
func (s *PostgresStorage) GetStocks() ([]Stock, error) {
	rows, err := s.db.Query("SELECT id, ticker, name FROM stocks")
	if err != nil {
		return nil, fmt.Errorf("error querying stocks: %w", err)
	}
	defer rows.Close()

	stocks := []Stock{}
	for rows.Next() {
		var stock Stock
		err := rows.Scan(&stock.ID, &stock.Ticker, &stock.Name)
		if err != nil {
			return nil, fmt.Errorf("error scanning stock: %w", err)
		}
		stocks = append(stocks, stock)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over stock rows: %w", err)
	}

	return stocks, nil
}

// GetPredictionsByTicker извлекает прогнозы для указанного тикера
func (s *PostgresStorage) GetPredictionsByTicker(ticker string) ([]Prediction, error) {
	var stockID int64
	err := s.db.QueryRow("SELECT id FROM stocks WHERE ticker = $1", ticker).Scan(&stockID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stock not found for ticker %s", ticker)
	} else if err != nil {
		return nil, fmt.Errorf("error getting stock ID for ticker %s: %w", ticker, err)
	}

	query := `
		SELECT
			p.message_id, p.stock_id, p.prediction_type,
			p.target_price, p.target_change_percent, p.period,
			p.recommendation, p.direction, p.justification_text,
			m.text, p.predicted_at
		FROM
			predictions p
		JOIN
			messages m ON p.message_id = m.telegram_id
		WHERE
			p.stock_id = $1
		ORDER BY
			p.predicted_at DESC
	`

	rows, err := s.db.Query(query, stockID)
	if err != nil {
		return nil, fmt.Errorf("error querying predictions: %w", err)
	}
	defer rows.Close()

	var counter int64 = 1
	predictions := []Prediction{}
	for rows.Next() {
		var p Prediction
		var predictedAt time.Time
		var messageText sql.NullString

		var temp int64
		err := rows.Scan(
			&temp, &p.StockID, &p.PredictionType,
			&p.TargetPrice, &p.TargetChangePercent, &p.Period,
			&p.Recommendation, &p.Direction, &p.JustificationText,
			&messageText, &predictedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning prediction: %w", err)
		}

		p.Message = &messageText.String
		p.MessageID = counter
		counter += 1
		p.PredictedAt = strconv.FormatInt(predictedAt.Unix(), 10) // Unix timestamp в строке
		predictions = append(predictions, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over prediction rows: %w", err)
	}

	return predictions, nil
}

// GetStockPriceHistory читает историю цен из CSV файла
func (s *PostgresStorage) GetStockPriceHistory(ticker string) ([]StockPriceHistory, error) {
	// Получаем StockID для тикера
	var stockID int64
	err := s.db.QueryRow("SELECT id FROM stocks WHERE ticker = $1", ticker).Scan(&stockID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stock not found for ticker %s", ticker)
	} else if err != nil {
		return nil, fmt.Errorf("error getting stock ID for ticker %s: %w", ticker, err)
	}

	// Путь к CSV файлу
	filename := fmt.Sprintf("%s_D1.csv", ticker)
	filepath := filepath.Join("data", filename)

	// Проверяем существование файла
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("price history file not found for ticker %s", ticker)
	}

	// Открываем CSV файл
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening price history file for ticker %s: %w", ticker, err)
	}
	defer file.Close()

	// Создаем CSV reader
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file for ticker %s: %w", ticker, err)
	}

	// Парсим данные
	var history []StockPriceHistory
	for i, record := range records {
		// Пропускаем заголовок (если есть)
		if i == 0 && strings.Contains(record[0], "Time") {
			continue
		}

		if len(record) < 8 {
			continue // Пропускаем некорректные строки
		}

		// Парсим время: "2025.09.15 00:00:00"
		timeStr := record[0]
		parsedTime, err := time.Parse("2006.01.02 15:04:05", timeStr)
		if err != nil {
			continue // Пропускаем строки с некорректной датой
		}

		// Парсим цену закрытия (Close)
		closePrice, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			continue // Пропускаем строки с некорректной ценой
		}

		// Парсим объем (RealVolume)
		volume, err := strconv.ParseInt(record[7], 10, 64)
		if err != nil {
			volume = 0 // Если не удалось распарсить объем, ставим 0
		}

		// Добавляем запись в историю
		history = append(history, StockPriceHistory{
			StockID:   stockID,
			Timestamp: parsedTime.Format(time.RFC3339), // ISO формат
			Price:     closePrice,
			Volume:    volume,
		})
	}

	// Сортируем по времени (от старых к новым)
	sort.Slice(history, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, history[i].Timestamp)
		timeJ, _ := time.Parse(time.RFC3339, history[j].Timestamp)
		return timeI.Before(timeJ)
	})

	return history, nil
}
