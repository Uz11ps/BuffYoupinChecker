package main

import (
	"log"
	"strconv"
	"time"

	"buff-youpin-checker/analyzer"
	"buff-youpin-checker/bot"
	"buff-youpin-checker/config"
	"buff-youpin-checker/database"
	"buff-youpin-checker/market"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()
	
	// Отладочная информация
	log.Printf("Подключение к БД: host=%s port=%s user=%s password=%s dbname=%s", 
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	// Подключаемся к базе данных
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	defer db.Close()

	// Создаем клиент для Market API
	marketClient := market.NewClient(cfg.MarketAPIKey)

	// Создаем анализатор трендов
	trendAnalyzer := analyzer.NewTrendAnalyzer(db)

	// Создаем бота
	telegramBot, err := bot.NewBot(cfg.TelegramToken, trendAnalyzer, db)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	// Запускаем сбор данных в отдельной горутине
	go startDataCollection(marketClient, db)

	// Запускаем анализ в отдельной горутине
	go startPeriodicAnalysis(trendAnalyzer)

	log.Println("🤖 Бот запущен и готов к работе!")
	
	// Запускаем бота (блокирующий вызов)
	telegramBot.Start()
}

// Периодический сбор данных с market.csgo.com
func startDataCollection(client *market.Client, db *database.DB) {
	ticker := time.NewTicker(10 * time.Minute) // Каждые 10 минут
	defer ticker.Stop()

	for {
		log.Println("📊 Собираю данные с market.csgo.com...")
		
		// Получаем текущие цены
		priceResponse, err := client.GetPrices()
		if err != nil {
			log.Printf("Ошибка получения цен: %v", err)
			<-ticker.C
			continue
		}

		if !priceResponse.Success {
			log.Println("API вернул ошибку в ответе")
			<-ticker.C
			continue
		}

		log.Printf("Получено %d предметов", len(priceResponse.Items))

		// Обрабатываем данные
		processedCount := 0
		for _, item := range priceResponse.Items {
			price := parseFloat(item.Price)
			if price <= 0 {
				continue
			}

			// Создаем или обновляем предмет
			dbItem := &database.Item{
				HashName:   item.MarketHashName,
				MarketName: item.MarketHashName,
				ClassID:    "unknown",
				InstanceID: "0",
			}

			err = db.CreateItem(dbItem)
			if err != nil {
				log.Printf("Ошибка создания предмета %s: %v", item.MarketHashName, err)
				continue
			}

			// Добавляем цену в историю
			err = db.AddPriceHistory(dbItem.ID, price, "RUB", "market.csgo.com")
			if err != nil {
				log.Printf("Ошибка добавления цены для %s: %v", item.MarketHashName, err)
				continue
			}

			processedCount++
		}

		log.Printf("✅ Обработано %d предметов", processedCount)
		<-ticker.C
	}
}

// Периодический анализ трендов
func startPeriodicAnalysis(analyzer *analyzer.TrendAnalyzer) {
	ticker := time.NewTicker(30 * time.Minute) // Каждые 30 минут
	defer ticker.Stop()

	// Первый анализ через 5 минут после запуска
	time.Sleep(5 * time.Minute)

	for {
		log.Println("🔍 Запускаю анализ трендов...")
		
		err := analyzer.AnalyzeAllItems()
		if err != nil {
			log.Printf("Ошибка анализа: %v", err)
		} else {
			log.Println("✅ Анализ трендов завершен")
		}

		<-ticker.C
	}
}

// Парсинг строки в float64
func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}