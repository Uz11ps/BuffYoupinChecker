package analyzer

import (
	"fmt"
	"math"
	"time"

	"buff-youpin-checker/database"
	"database/sql"
)

type TrendAnalyzer struct {
	db *database.DB
}

type ItemTrend struct {
	ItemID         int     `json:"item_id"`
	HashName       string  `json:"hash_name"`
	MarketName     string  `json:"market_name"`
	Category       string  `json:"category"`
	ImageURL       string  `json:"image_url"`
	CurrentPrice   float64 `json:"current_price"`
	GrowthRate     float64 `json:"growth_rate"`     // процент роста за период
	Volatility     float64 `json:"volatility"`      // волатильность цены
	TrendScore     int     `json:"trend_score"`     // рейтинг от 1 до 10
	Recommendation string  `json:"recommendation"`  // BUY/HOLD/SELL
	PredictedGrowth float64 `json:"predicted_growth"` // прогнозируемый рост
	ExpectedROI    float64 `json:"expected_roi"`    // ожидаемый ROI (множитель)
	Price          float64 `json:"price"`           // цена для расчетов
}

func NewTrendAnalyzer(db *database.DB) *TrendAnalyzer {
	return &TrendAnalyzer{db: db}
}

// Анализ трендов для всех предметов
func (ta *TrendAnalyzer) AnalyzeAllItems() error {
	// Получаем все предметы с историей цен
	query := `SELECT DISTINCT i.id, i.hash_name, i.market_name 
			  FROM items i 
			  INNER JOIN price_history ph ON i.id = ph.item_id`
	
	rows, err := ta.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID int
		var hashName, marketName string
		
		if err := rows.Scan(&itemID, &hashName, &marketName); err != nil {
			continue
		}

		trend, err := ta.analyzeItemTrend(itemID, hashName, marketName)
		if err != nil {
			continue
		}

		// Сохраняем результат анализа
		ta.saveAnalysis(trend)
	}

	return nil
}

// Анализ тренда для конкретного предмета
func (ta *TrendAnalyzer) analyzeItemTrend(itemID int, hashName, marketName string) (*ItemTrend, error) {
	// Получаем историю цен за последние 30 дней
	query := `SELECT price, recorded_at FROM price_history 
			  WHERE item_id = $1 AND recorded_at >= $2 
			  ORDER BY recorded_at ASC`
	
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	rows, err := ta.db.Query(query, itemID, thirtyDaysAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []float64
	var timestamps []time.Time
	
	for rows.Next() {
		var price float64
		var timestamp time.Time
		
		if err := rows.Scan(&price, &timestamp); err != nil {
			continue
		}
		
		prices = append(prices, price)
		timestamps = append(timestamps, timestamp)
	}

	if len(prices) < 1 {
		return nil, fmt.Errorf("insufficient price data")
	}

	// Рассчитываем метрики
	currentPrice := prices[len(prices)-1]
	var growthRate float64
	if len(prices) > 1 {
		oldPrice := prices[0]
		growthRate = ((currentPrice - oldPrice) / oldPrice) * 100
	} else {
		// Для новых предметов используем базовый рост на основе цены
		if currentPrice > 100 {
			growthRate = 5.0 // Дорогие предметы - потенциал роста
		} else if currentPrice > 10 {
			growthRate = 10.0 // Средние цены - больший потенциал
		} else {
			growthRate = 15.0 // Дешевые предметы - максимальный потенциал
		}
	}
	volatility := ta.calculateVolatility(prices)
	trendScore := ta.calculateTrendScore(growthRate, volatility, len(prices))
	recommendation := ta.getRecommendation(trendScore, growthRate)
	predictedGrowth := ta.predictGrowth(prices, timestamps)

	return &ItemTrend{
		ItemID:          itemID,
		HashName:        hashName,
		MarketName:      marketName,
		CurrentPrice:    currentPrice,
		GrowthRate:      growthRate,
		Volatility:      volatility,
		TrendScore:      trendScore,
		Recommendation:  recommendation,
		PredictedGrowth: predictedGrowth,
	}, nil
}

// Расчет волатильности цены
func (ta *TrendAnalyzer) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// Среднее значение
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	// Дисперсия
	variance := 0.0
	for _, price := range prices {
		variance += math.Pow(price-mean, 2)
	}
	variance /= float64(len(prices))

	// Коэффициент вариации (волатильность)
	return (math.Sqrt(variance) / mean) * 100
}

// Расчет рейтингового скора (1-10)
func (ta *TrendAnalyzer) calculateTrendScore(growthRate, volatility float64, dataPoints int) int {
	score := 5.0 // базовый скор

	// Положительный рост увеличивает скор
	if growthRate > 0 {
		score += math.Min(growthRate/10, 3) // максимум +3 за рост
	} else {
		score += math.Max(growthRate/20, -3) // максимум -3 за падение
	}

	// Низкая волатильность увеличивает скор
	if volatility < 10 {
		score += 1
	} else if volatility > 30 {
		score -= 1
	}

	// Больше данных = надежнее
	if dataPoints > 20 {
		score += 0.5
	}

	// Ограничиваем скор от 1 до 10
	if score < 1 {
		score = 1
	} else if score > 10 {
		score = 10
	}

	return int(math.Round(score))
}

// Получение рекомендации
func (ta *TrendAnalyzer) getRecommendation(trendScore int, growthRate float64) string {
	if trendScore >= 8 && growthRate > 5 {
		return "BUY"
	} else if trendScore >= 6 {
		return "HOLD"
	} else {
		return "SELL"
	}
}

// Простой прогноз роста на основе линейного тренда
func (ta *TrendAnalyzer) predictGrowth(prices []float64, timestamps []time.Time) float64 {
	if len(prices) < 3 {
		return 0
	}

	// Простая линейная регрессия для прогноза
	n := float64(len(prices))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumX2 += x * x
	}

	// Коэффициент наклона
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	// Прогнозируем на 7 дней вперед
	currentPrice := prices[len(prices)-1]
	futurePrice := currentPrice + slope*7
	
	return ((futurePrice - currentPrice) / currentPrice) * 100
}

// Сохранение результатов анализа
func (ta *TrendAnalyzer) saveAnalysis(trend *ItemTrend) error {
	query := `INSERT INTO item_analysis (item_id, growth_rate, volatility, trend_score, recommendation) 
			  VALUES ($1, $2, $3, $4, $5)
			  ON CONFLICT (item_id) DO UPDATE SET
			  growth_rate = $2, volatility = $3, trend_score = $4, 
			  recommendation = $5, analysis_date = CURRENT_TIMESTAMP`
			  
	_, err := ta.db.Exec(query, trend.ItemID, trend.GrowthRate, 
		trend.Volatility, trend.TrendScore, trend.Recommendation)
	return err
}

// Получение топовых предметов по рейтингу
func (ta *TrendAnalyzer) GetTopItems(limit int) ([]ItemTrend, error) {
	query := `SELECT ia.item_id, i.hash_name, i.market_name, i.category, i.image_url,
			  ia.growth_rate, ia.volatility, ia.trend_score, ia.recommendation,
			  (SELECT price FROM price_history WHERE item_id = ia.item_id ORDER BY recorded_at DESC LIMIT 1) as current_price
			  FROM item_analysis ia
			  JOIN items i ON ia.item_id = i.id
			  WHERE ia.trend_score >= 6
			  ORDER BY ia.trend_score DESC, ia.growth_rate DESC
			  LIMIT $1`

	rows, err := ta.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []ItemTrend
	for rows.Next() {
		var trend ItemTrend
		var currentPrice sql.NullFloat64
		
		err := rows.Scan(&trend.ItemID, &trend.HashName, &trend.MarketName, 
			&trend.Category, &trend.ImageURL, &trend.GrowthRate, &trend.Volatility, 
			&trend.TrendScore, &trend.Recommendation, &currentPrice)
		if err != nil {
			continue
		}
		
		if currentPrice.Valid {
			trend.CurrentPrice = currentPrice.Float64
			trend.Price = currentPrice.Float64  // Для расчетов бюджета
		}
		
		trends = append(trends, trend)
	}

	return trends, nil
}

// Получить лучшие предметы для инвестиций с минимальным ROI
func (ta *TrendAnalyzer) GetBestInvestmentItems(limit int, minROI float64) ([]ItemTrend, error) {
	query := `SELECT ia.item_id, i.hash_name, i.market_name, i.category, i.image_url,
			  ia.growth_rate, ia.volatility, ia.trend_score, ia.recommendation,
			  (SELECT price FROM price_history WHERE item_id = ia.item_id ORDER BY recorded_at DESC LIMIT 1) as current_price
			  FROM item_analysis ia
			  JOIN items i ON ia.item_id = i.id
			  WHERE ia.trend_score >= 6 
			    AND ia.recommendation = 'BUY'
			    AND (1 + ia.growth_rate/100.0) >= $1
			  ORDER BY ia.trend_score DESC, ia.growth_rate DESC
			  LIMIT $2`

	rows, err := ta.db.Query(query, minROI, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []ItemTrend
	for rows.Next() {
		var trend ItemTrend
		var currentPrice sql.NullFloat64
		
		err := rows.Scan(&trend.ItemID, &trend.HashName, &trend.MarketName, 
			&trend.Category, &trend.ImageURL, &trend.GrowthRate, &trend.Volatility, 
			&trend.TrendScore, &trend.Recommendation, &currentPrice)
		if err != nil {
			continue
		}
		
		if currentPrice.Valid {
			trend.CurrentPrice = currentPrice.Float64
			trend.Price = currentPrice.Float64  // Для расчетов бюджета
		}
		
		// Рассчитываем ожидаемый ROI
		trend.ExpectedROI = 1 + (trend.GrowthRate / 100.0)
		
		trends = append(trends, trend)
	}

	return trends, nil
}

func (ta *TrendAnalyzer) GetTopItemsByCategory(category string, limit int) ([]ItemTrend, error) {
	query := `SELECT ia.item_id, i.hash_name, i.market_name, i.category, i.image_url,
			  ia.growth_rate, ia.volatility, ia.trend_score, ia.recommendation,
			  (SELECT price FROM price_history WHERE item_id = ia.item_id ORDER BY recorded_at DESC LIMIT 1) as current_price
			  FROM item_analysis ia
			  JOIN items i ON ia.item_id = i.id
			  WHERE i.category = $1 AND ia.trend_score >= 6
			  ORDER BY ia.trend_score DESC, ia.growth_rate DESC
			  LIMIT $2`

	rows, err := ta.db.Query(query, category, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []ItemTrend
	for rows.Next() {
		var trend ItemTrend
		var currentPrice sql.NullFloat64
		
		err := rows.Scan(&trend.ItemID, &trend.HashName, &trend.MarketName, 
			&trend.Category, &trend.ImageURL, &trend.GrowthRate, &trend.Volatility, 
			&trend.TrendScore, &trend.Recommendation, &currentPrice)
		if err != nil {
			continue
		}
		
		if currentPrice.Valid {
			trend.CurrentPrice = currentPrice.Float64
			trend.Price = currentPrice.Float64  // Для расчетов бюджета
		}
		
		trends = append(trends, trend)
	}

	return trends, nil
}