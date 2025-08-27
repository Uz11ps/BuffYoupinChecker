package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"buff-youpin-checker/config"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type Item struct {
	ID         int       `json:"id"`
	HashName   string    `json:"hash_name"`
	MarketName string    `json:"market_name"`
	ClassID    string    `json:"class_id"`
	InstanceID string    `json:"instance_id"`
	Category   string    `json:"category"`
	ImageURL   string    `json:"image_url"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PriceHistory struct {
	ID         int       `json:"id"`
	ItemID     int       `json:"item_id"`
	Price      float64   `json:"price"`
	Currency   string    `json:"currency"`
	RecordedAt time.Time `json:"recorded_at"`
	Source     string    `json:"source"`
}

type ItemAnalysis struct {
	ID             int       `json:"id"`
	ItemID         int       `json:"item_id"`
	GrowthRate     float64   `json:"growth_rate"`
	Volatility     float64   `json:"volatility"`
	TrendScore     int       `json:"trend_score"`
	Recommendation string    `json:"recommendation"`
	AnalysisDate   time.Time `json:"analysis_date"`
}

func Connect(cfg *config.Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) CreateItem(item *Item) error {
	// Определяем категорию и URL изображения
	item.Category = determineCategory(item.MarketName)
	item.ImageURL = generateImageURL(item.MarketName)
	
	query := `INSERT INTO items (hash_name, market_name, class_id, instance_id, category, image_url) 
			  VALUES ($1, $2, $3, $4, $5, $6) 
			  ON CONFLICT (hash_name) DO UPDATE SET 
			  market_name = $2, category = $5, image_url = $6, updated_at = CURRENT_TIMESTAMP
			  RETURNING id`
	
	return db.QueryRow(query, item.HashName, item.MarketName, item.ClassID, item.InstanceID, item.Category, item.ImageURL).Scan(&item.ID)
}

// Определение категории предмета по названию
func determineCategory(marketName string) string {
	name := strings.ToLower(marketName)
	
	// Ножи - полный список CS2
	if strings.Contains(name, "knife") || strings.Contains(name, "bayonet") || 
	   strings.Contains(name, "karambit") || strings.Contains(name, "butterfly") ||
	   strings.Contains(name, "flip") || strings.Contains(name, "gut") ||
	   strings.Contains(name, "huntsman") || strings.Contains(name, "falchion") ||
	   strings.Contains(name, "bowie") || strings.Contains(name, "shadow") ||
	   strings.Contains(name, "daggers") || strings.Contains(name, "ursus") ||
	   strings.Contains(name, "navaja") || strings.Contains(name, "stiletto") ||
	   strings.Contains(name, "talon") || strings.Contains(name, "survival") ||
	   strings.Contains(name, "nomad") || strings.Contains(name, "skeleton") ||
	   strings.Contains(name, "paracord") || strings.Contains(name, "kukri") ||
	   strings.Contains(name, "m9 bayonet") || strings.Contains(name, "classic knife") {
		return "knives"
	}
	
	// Контейнеры - только кейсы (исключаем скины с "case" в названии)
	if strings.Contains(name, "case") && !strings.Contains(name, "key") && 
	   !strings.Contains(name, "case hardened") && !strings.Contains(name, "|") {
		return "containers"
	}
	
	// Ключи - отдельная категория
	if strings.Contains(name, "key") {
		return "keys"
	}
	
	// Капсулы, наборы, пакеты - отдельная категория
	if strings.Contains(name, "capsule") || strings.Contains(name, "package") ||
	   strings.Contains(name, "souvenir") {
		return "packages"
	}
	
	// Брелки
	if strings.Contains(name, "charm") || strings.Contains(name, "keychain") {
		return "charms"
	}
	
	// Стикеры
	if strings.Contains(name, "sticker") {
		return "stickers"
	}
	
	// Перчатки
	if strings.Contains(name, "gloves") {
		return "gloves"
	}
	
	// Оружие (по умолчанию)
	return "weapons"
}

// Генерация URL изображения Steam
func generateImageURL(marketName string) string {
	// Steam CDN URL для изображений предметов
	baseURL := "https://steamcommunity-a.akamaihd.net/economy/image/"
	// Для демонстрации используем заглушку
	return baseURL + "placeholder"
}

func (db *DB) AddPriceHistory(itemID int, price float64, currency, source string) error {
	query := `INSERT INTO price_history (item_id, price, currency, source) VALUES ($1, $2, $3, $4)`
	_, err := db.Exec(query, itemID, price, currency, source)
	return err
}

func (db *DB) GetTopTrendingItems(limit int) ([]ItemAnalysis, error) {
	query := `SELECT ia.item_id, ia.growth_rate, ia.volatility, ia.trend_score, 
			  ia.recommendation, i.hash_name, i.market_name
			  FROM item_analysis ia
			  JOIN items i ON ia.item_id = i.id
			  ORDER BY ia.trend_score DESC
			  LIMIT $1`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ItemAnalysis
	for rows.Next() {
		var analysis ItemAnalysis
		var hashName, marketName string
		
		err := rows.Scan(&analysis.ItemID, &analysis.GrowthRate, &analysis.Volatility,
			&analysis.TrendScore, &analysis.Recommendation, &hashName, &marketName)
		if err != nil {
			continue
		}
		
		results = append(results, analysis)
	}

	return results, nil
}