package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"buff-youpin-checker/analyzer"
	"buff-youpin-checker/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	analyzer *analyzer.TrendAnalyzer
	db       *database.DB
}

func NewBot(token string, analyzer *analyzer.TrendAnalyzer, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	api.Debug = false
	log.Printf("Бот авторизован как %s", api.Self.UserName)

	return &Bot{
		api:      api,
		analyzer: analyzer,
		db:       db,
	}, nil
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
		}
	}
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendWelcomeMessage(message.Chat.ID)
	case "top":
		b.sendTopItems(message.Chat.ID)
	case "budget":
		b.sendBudgetCalculator(message.Chat.ID)
	case "analyze":
		b.runAnalysis(message.Chat.ID)
	default:
		if message.IsCommand() {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Неизвестная команда. Используйте /start для помощи.")
			b.api.Send(msg)
		} else {
			// Пробуем парсить как бюджет
			if budget, err := strconv.ParseFloat(message.Text, 64); err == nil && budget > 0 {
				if budget < 1000 {
					msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Минимальный бюджет: 1000₽")
					b.api.Send(msg)
				} else if budget > 10000000 {
					msg := tgbotapi.NewMessage(message.Chat.ID, "❌ Максимальный бюджет: 10,000,000₽")
					b.api.Send(msg)
				} else {
					b.sendBudgetResults(message.Chat.ID, budget)
				}
			}
		}
	}
}

func (b *Bot) sendWelcomeMessage(chatID int64) {
	text := `🎮 *Добро пожаловать в CS2 Skin Analyzer!*

Этот бот поможет вам найти перспективные скины для инвестиций.

📊 *Доступные команды:*
/top - Топ перспективных скинов (с пагинацией)
/budget - Рассчитать оптимальный портфель инвестиций
/analyze - Запустить анализ рынка

🚀 *Как это работает:*
Бот анализирует ценовые тренды скинов и выдает рейтинг от 1 до 10, где 10 - максимально перспективный предмет для покупки.

💡 *Рекомендации:*
• 🟢 BUY - рекомендуется к покупке
• 🟡 HOLD - держать если есть  
• 🔴 SELL - рекомендуется продать

🔍 *Навигация:*
• Нажмите на предмет для детального анализа
• Используйте ⬅️➡️ для перехода между страницами`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) sendTopItems(chatID int64) {
	// Показываем меню категорий
	text := "📂 *Выберите категорию для анализа:*\n\n"
	text += "🔪 Ножи - самые дорогие и стабильные инвестиции\n"
	text += "🔫 Оружие - популярные скины с хорошим потенциалом\n"
	text += "📦 Кейсы - контейнеры со скинами (только Case)\n"
	text += "🧤 Перчатки - редкие и ценные предметы\n"
	text += "🗝️ Ключи - для открытия кейсов\n"
	text += "📤 Пакеты - капсулы и сувениры\n"
	text += "🏷️ Стикеры - коллекционная ценность\n"
	text += "🎯 Брелки - новая категория предметов\n"
	text += "⭐ Все категории - общий топ"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔪 Ножи", "cat_knives"),
			tgbotapi.NewInlineKeyboardButtonData("🔫 Оружие", "cat_weapons"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Кейсы", "cat_containers"),
			tgbotapi.NewInlineKeyboardButtonData("🧤 Перчатки", "cat_gloves"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗝️ Ключи", "cat_keys"),
			tgbotapi.NewInlineKeyboardButtonData("📤 Пакеты", "cat_packages"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏷️ Стикеры", "cat_stickers"),
			tgbotapi.NewInlineKeyboardButtonData("🎯 Брелки", "cat_charms"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⭐ Все категории", "cat_all"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendTopItemsPage(chatID int64, page int) {
	b.sendTopItemsByCategory(chatID, "all", page)
}

func (b *Bot) sendTopItemsByCategory(chatID int64, category string, page int) {
	itemsPerPage := 5
	offset := (page - 1) * itemsPerPage
	
	// Получаем предметы по категории
	var allTrends []analyzer.ItemTrend
	var err error
	
	if category == "all" {
		allTrends, err = b.analyzer.GetTopItems(50)
	} else {
		allTrends, err = b.analyzer.GetTopItemsByCategory(category, 50)
	}
	
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка получения данных. Попробуйте позже.")
		b.api.Send(msg)
		return
	}

	if len(allTrends) == 0 {
		msg := tgbotapi.NewMessage(chatID, "В данной категории пока нет анализируемых предметов. Попробуйте /analyze для обновления.")
		b.api.Send(msg)
		return
	}

	// Вычисляем общее количество страниц
	totalPages := (len(allTrends) + itemsPerPage - 1) / itemsPerPage
	if page > totalPages {
		page = totalPages
	}
	if page < 1 {
		page = 1
	}

	// Получаем предметы для текущей страницы
	start := offset
	end := offset + itemsPerPage
	if end > len(allTrends) {
		end = len(allTrends)
	}
	
	trends := allTrends[start:end]

	categoryName := b.getCategoryName(category)
	text := fmt.Sprintf("🏆 *%s* (стр. %d/%d)\n\n", categoryName, page, totalPages)
	
	for i, trend := range trends {
		emoji := b.getRecommendationEmoji(trend.Recommendation)
		catEmoji := b.getCategoryEmoji(trend.Category)
		globalIndex := start + i + 1
		
		text += fmt.Sprintf("%d. %s %s %s\n", globalIndex, emoji, catEmoji, trend.MarketName)
		text += fmt.Sprintf("   📊 Рейтинг: %d/10 | 💰 %.2f ₽ | 📈 %.1f%%\n", 
			trend.TrendScore, trend.CurrentPrice, trend.GrowthRate)
		text += fmt.Sprintf("   💡 %s\n\n", b.getInvestmentAdvice(trend))
	}

	// Создаем клавиатуру с предметами
	var keyboard [][]tgbotapi.InlineKeyboardButton
	
	for _, trend := range trends {
		buttonText := fmt.Sprintf("📊 %s", b.truncateString(trend.MarketName, 30))
		callbackData := fmt.Sprintf("item_%d", trend.ItemID)
		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	// Добавляем кнопки навигации
	var navButtons []tgbotapi.InlineKeyboardButton
	
	if page > 1 {
		prevCallback := fmt.Sprintf("page_%d_%s", page-1, category)
		prevButton := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", prevCallback)
		navButtons = append(navButtons, prevButton)
	}
	
	if page < totalPages {
		nextCallback := fmt.Sprintf("page_%d_%s", page+1, category)
		nextButton := tgbotapi.NewInlineKeyboardButtonData("Вперед ➡️", nextCallback)
		navButtons = append(navButtons, nextButton)
	}
	
	if len(navButtons) > 0 {
		keyboard = append(keyboard, navButtons)
	}

	// Кнопка возврата к категориям
	backButton := tgbotapi.NewInlineKeyboardButtonData("📂 Выбрать категорию", "back_to_top")
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{backButton})

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	b.api.Send(msg)
}

func (b *Bot) runAnalysis(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🔄 Запускаю анализ рынка... Это может занять несколько минут.")
	b.api.Send(msg)

	go func() {
		err := b.analyzer.AnalyzeAllItems()
		if err != nil {
			errorMsg := tgbotapi.NewMessage(chatID, "❌ Ошибка при анализе: "+err.Error())
			b.api.Send(errorMsg)
			return
		}

		successMsg := tgbotapi.NewMessage(chatID, "✅ Анализ завершен! Используйте /top для просмотра результатов.")
		b.api.Send(successMsg)
	}()
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	// Отвечаем на callback чтобы убрать "часики"
	b.api.Request(tgbotapi.NewCallback(callback.ID, ""))

	if callback.Data == "back_to_top" {
		b.sendTopItems(callback.Message.Chat.ID)
		return
	}

	if len(callback.Data) > 4 && callback.Data[:4] == "cat_" {
		category := callback.Data[4:]
		b.sendTopItemsByCategory(callback.Message.Chat.ID, category, 1)
		return
	}

	if len(callback.Data) > 5 && callback.Data[:5] == "item_" {
		itemIDStr := callback.Data[5:]
		itemID, err := strconv.Atoi(itemIDStr)
		if err != nil {
			return
		}

		b.sendItemDetails(callback.Message.Chat.ID, itemID)
		return
	}

	if len(callback.Data) > 5 && callback.Data[:5] == "page_" {
		parts := strings.Split(callback.Data[5:], "_")
		if len(parts) >= 1 {
			page, err := strconv.Atoi(parts[0])
			if err != nil {
				return
			}
			
			if len(parts) >= 2 {
				category := parts[1]
				b.sendTopItemsByCategory(callback.Message.Chat.ID, category, page)
			} else {
				b.sendTopItemsPage(callback.Message.Chat.ID, page)
			}
		}
		return
	}

	// Обработка кнопок бюджета
	if len(callback.Data) > 7 && callback.Data[:7] == "budget_" {
		action := callback.Data[7:]
		
		switch action {
		case "5000":
			b.sendBudgetResults(callback.Message.Chat.ID, 5000)
		case "10000":
			b.sendBudgetResults(callback.Message.Chat.ID, 10000)
		case "25000":
			b.sendBudgetResults(callback.Message.Chat.ID, 25000)
		case "50000":
			b.sendBudgetResults(callback.Message.Chat.ID, 50000)
		case "100000":
			b.sendBudgetResults(callback.Message.Chat.ID, 100000)
		case "custom":
			msg := tgbotapi.NewMessage(callback.Message.Chat.ID, 
				"💰 Введите ваш бюджет числом в рублях:\nНапример: 15000")
			b.api.Send(msg)
		case "new":
			b.sendBudgetCalculator(callback.Message.Chat.ID)
		}
		return
	}
}

func (b *Bot) sendItemDetails(chatID int64, itemID int) {
	// Получаем детальную информацию о предмете
	query := `SELECT i.hash_name, i.market_name, i.category, i.image_url, 
			  ia.growth_rate, ia.volatility, ia.trend_score, ia.recommendation,
			  (SELECT price FROM price_history WHERE item_id = $1 ORDER BY recorded_at DESC LIMIT 1) as current_price,
			  (SELECT COUNT(*) FROM price_history WHERE item_id = $1) as data_points
			  FROM items i
			  JOIN item_analysis ia ON i.id = ia.item_id
			  WHERE i.id = $1`

	var hashName, marketName, category, imageURL, recommendation string
	var growthRate, volatility, currentPrice float64
	var trendScore, dataPoints int

	err := b.db.QueryRow(query, itemID).Scan(&hashName, &marketName, &category, &imageURL,
		&growthRate, &volatility, &trendScore, &recommendation, &currentPrice, &dataPoints)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Ошибка получения информации о предмете.")
		b.api.Send(msg)
		return
	}

	emoji := b.getRecommendationEmoji(recommendation)
	catEmoji := b.getCategoryEmoji(category)
	
	text := fmt.Sprintf("📊 *Подробный инвестиционный анализ*\n\n")
	text += fmt.Sprintf("%s *%s*\n", catEmoji, marketName)
	text += fmt.Sprintf("📂 Категория: %s\n\n", b.getCategoryName(category))
	
	text += fmt.Sprintf("💰 *Цена:* %.2f ₽\n", currentPrice)
	text += fmt.Sprintf("📈 *Рост:* %.1f%% за период\n", growthRate)
	text += fmt.Sprintf("📊 *Волатильность:* %.1f%%\n", volatility)
	text += fmt.Sprintf("⭐ *Рейтинг:* %d/10\n", trendScore)
	text += fmt.Sprintf("%s *Рекомендация:* %s\n\n", emoji, recommendation)

	// Детальная интерпретация
	text += "🔍 *Почему стоит рассмотреть:*\n"
	text += b.getDetailedAnalysis(trendScore, growthRate, currentPrice, category, volatility)
	
	text += "\n📈 *Инвестиционная стратегия:*\n"
	text += b.getInvestmentStrategy(trendScore, recommendation, currentPrice, category)
	
	text += fmt.Sprintf("\n📊 *Надежность данных:* %d точек\n", dataPoints)

	// Кнопка для возврата к списку
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к списку", "back_to_top"),
		),
	)

	// Если есть валидный URL изображения, отправляем фото
	if imageURL != "" && imageURL != "https://steamcommunity-a.akamaihd.net/economy/image/placeholder" {
		photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(imageURL))
		photo.Caption = text
		photo.ParseMode = "Markdown"
		photo.ReplyMarkup = keyboard
		b.api.Send(photo)
	} else {
		// Отправляем текстовое сообщение если нет изображения
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
}

func (b *Bot) getRecommendationEmoji(recommendation string) string {
	switch recommendation {
	case "BUY":
		return "🟢"
	case "HOLD":
		return "🟡"
	case "SELL":
		return "🔴"
	default:
		return "⚪"
	}
}

func (b *Bot) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func (b *Bot) getCategoryName(category string) string {
	switch category {
	case "knives":
		return "🔪 ТОП Ножей"
	case "weapons":
		return "🔫 ТОП Оружия"
	case "containers":
		return "📦 ТОП Кейсов"
	case "keys":
		return "🗝️ ТОП Ключей"
	case "packages":
		return "📤 ТОП Пакетов"
	case "gloves":
		return "🧤 ТОП Перчаток"
	case "stickers":
		return "🏷️ ТОП Стикеров"
	case "charms":
		return "🎯 ТОП Брелков"
	default:
		return "⭐ ТОП Всех категорий"
	}
}

func (b *Bot) getCategoryEmoji(category string) string {
	switch category {
	case "knives":
		return "🔪"
	case "weapons":
		return "🔫"
	case "containers":
		return "📦"
	case "keys":
		return "🗝️"
	case "packages":
		return "📤"
	case "gloves":
		return "🧤"
	case "stickers":
		return "🏷️"
	case "charms":
		return "🎯"
	default:
		return "⚡"
	}
}

func (b *Bot) getInvestmentAdvice(trend analyzer.ItemTrend) string {
	if trend.TrendScore >= 8 && trend.GrowthRate > 10 {
		return "Сильная покупка - отличный потенциал роста!"
	} else if trend.TrendScore >= 7 && trend.CurrentPrice < 10 {
		return "Дешевый актив с хорошими перспективами"
	} else if trend.TrendScore >= 7 && trend.CurrentPrice > 100 {
		return "Стабильная дорогая инвестиция"
	} else if trend.TrendScore >= 6 {
		return "Умеренный потенциал, подходит для диверсификации"
	} else {
		return "Рискованная инвестиция, требует анализа"
	}
}

func (b *Bot) getDetailedAnalysis(trendScore int, growthRate, currentPrice float64, category string, volatility float64) string {
	analysis := ""
	
	// Анализ по рейтингу
	if trendScore >= 8 {
		analysis += "✅ Высокий рейтинг указывает на сильные перспективы роста\n"
	} else if trendScore >= 6 {
		analysis += "⚡ Средний рейтинг - предмет имеет потенциал\n"
	}
	
	// Анализ по росту
	if growthRate > 15 {
		analysis += "🚀 Демонстрирует активный рост цены\n"
	} else if growthRate > 5 {
		analysis += "📈 Стабильный положительный тренд\n"
	}
	
	// Анализ по цене
	if currentPrice < 10 {
		analysis += "💎 Низкая цена входа - минимальный риск\n"
	} else if currentPrice > 100 {
		analysis += "💰 Premium сегмент - для серьезных инвесторов\n"
	}
	
	// Анализ по категории
	switch category {
	case "knives":
		analysis += "🔪 Ножи - самая стабильная категория для долгосрочных инвестиций\n"
	case "weapons":
		analysis += "🔫 Оружие - высокая ликвидность и спрос\n"
	case "containers":
		analysis += "📦 Контейнеры - растут в цене со временем\n"
	case "gloves":
		analysis += "🧤 Перчатки - редкая категория с ограниченным предложением\n"
	}
	
	// Анализ волатильности
	if volatility < 10 {
		analysis += "🛡️ Стабильная цена - низкий риск потерь\n"
	} else if volatility > 30 {
		analysis += "⚡ Высокая волатильность - возможны быстрые изменения\n"
	}
	
	return analysis
}

func (b *Bot) getInvestmentStrategy(trendScore int, recommendation string, currentPrice float64, category string) string {
	strategy := ""
	
	switch recommendation {
	case "BUY":
		strategy += "🟢 *Покупать сейчас* - оптимальная точка входа\n"
		if currentPrice < 50 {
			strategy += "💡 Можно купить несколько штук для диверсификации\n"
		}
		strategy += "⏰ Рекомендуемый срок холда: 3-6 месяцев\n"
		
	case "HOLD":
		strategy += "🟡 *Держать* - если уже есть в портфеле\n"
		strategy += "📊 Следить за динамикой, возможна покупка при снижении\n"
		
	case "SELL":
		strategy += "🔴 *Продавать* - высокий риск снижения\n"
		strategy += "💸 Рассмотреть фиксацию прибыли если есть\n"
	}
	
	// Специфичные стратегии по категориям
	switch category {
	case "knives":
		strategy += "🔪 Ножи лучше покупать в хорошем состоянии (MW, FN)\n"
	case "containers":
		strategy += "📦 Контейнеры - долгосрочная игра, держать минимум год\n"
	case "weapons":
		strategy += "🔫 Популярное оружие (AK, M4, AWP) предпочтительнее\n"
	}
	
	return strategy
}

// Калькулятор бюджета
func (b *Bot) sendBudgetCalculator(chatID int64) {
	text := `💰 *Калькулятор бюджета*

Рассчитаем оптимальный портфель для вашего капитала с минимальной доходностью **210%**!

🎯 *Что делает калькулятор:*
• Анализирует топ предметы с лучшими прогнозами
• Распределяет ваш бюджет между перспективными скинами
• Учитывает риски и диверсификацию
• Показывает ожидаемую прибыль через 6-12 месяцев

💡 *Введите ваш бюджет в рублях:*
Например: 10000`

	// Создаем клавиатуру с примерами бюджетов
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("5 000₽", "budget_5000"),
			tgbotapi.NewInlineKeyboardButtonData("10 000₽", "budget_10000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("25 000₽", "budget_25000"),
			tgbotapi.NewInlineKeyboardButtonData("50 000₽", "budget_50000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("100 000₽", "budget_100000"),
			tgbotapi.NewInlineKeyboardButtonData("💬 Ввести свой", "budget_custom"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// Структура для рекомендации покупки
type BudgetRecommendation struct {
	ItemName        string
	Category        string
	Price           float64
	Quantity        int
	TotalCost       float64
	ExpectedROI     float64
	ExpectedProfit  float64
	TrendScore      int
	Recommendation  string
}

// Расчет оптимального портфеля
func (b *Bot) calculateOptimalPortfolio(budget float64) ([]BudgetRecommendation, float64, error) {
	// Получаем топ предметы с минимальным ROI 210%
	items, err := b.analyzer.GetBestInvestmentItems(50, 2.1) // минимум 210% ROI
	if err != nil {
		return nil, 0, err
	}

	var recommendations []BudgetRecommendation
	remainingBudget := budget
	totalExpectedProfit := 0.0

	// Распределяем бюджет по категориям (диверсификация)
	categoryAllocations := map[string]float64{
		"knives":     0.4,  // 40% на ножи (стабильно)
		"weapons":    0.3,  // 30% на оружие (ликвидно)
		"containers": 0.15, // 15% на кейсы (долгосрочно)
		"gloves":     0.1,  // 10% на перчатки (премиум)
		"stickers":   0.05, // 5% на стикеры (высокий риск)
	}

	for category, allocation := range categoryAllocations {
		categoryBudget := budget * allocation
		categoryItems := filterItemsByCategory(items, category)
		
		for _, item := range categoryItems {
			if remainingBudget < item.Price || categoryBudget < item.Price {
				continue
			}

			// Рассчитываем количество предметов для покупки
			maxQuantity := int(categoryBudget / item.Price)
			if maxQuantity > 3 {
				maxQuantity = 3 // Ограничиваем максимум 3 штуки одного предмета
			}
			if maxQuantity == 0 {
				continue
			}

			totalCost := item.Price * float64(maxQuantity)
			expectedProfit := totalCost * (item.ExpectedROI - 1.0)

			recommendation := BudgetRecommendation{
				ItemName:       item.MarketName,
				Category:       item.Category,
				Price:          item.Price,
				Quantity:       maxQuantity,
				TotalCost:      totalCost,
				ExpectedROI:    item.ExpectedROI,
				ExpectedProfit: expectedProfit,
				TrendScore:     item.TrendScore,
				Recommendation: item.Recommendation,
			}

			recommendations = append(recommendations, recommendation)
			remainingBudget -= totalCost
			categoryBudget -= totalCost
			totalExpectedProfit += expectedProfit

			// Ограничиваем количество рекомендаций
			if len(recommendations) >= 8 {
				break
			}
		}
	}

	return recommendations, totalExpectedProfit, nil
}

// Отправляем результаты расчета бюджета
func (b *Bot) sendBudgetResults(chatID int64, budget float64) {
	recommendations, totalExpectedProfit, err := b.calculateOptimalPortfolio(budget)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "❌ Ошибка при расчете портфеля. Попробуйте позже.")
		b.api.Send(msg)
		return
	}

	if len(recommendations) == 0 {
		msg := tgbotapi.NewMessage(chatID, "❌ Не найдено подходящих предметов для вашего бюджета с требуемой доходностью 210%+")
		b.api.Send(msg)
		return
	}

	text := fmt.Sprintf("💰 *Оптимальный портфель для %s₽*\n\n", formatPrice(budget))

	totalInvested := 0.0
	for _, rec := range recommendations {
		totalInvested += rec.TotalCost
	}

	expectedROI := ((totalInvested + totalExpectedProfit) / totalInvested) * 100
	
	text += fmt.Sprintf("📊 *Общая статистика:*\n")
	text += fmt.Sprintf("💵 К инвестированию: %s₽\n", formatPrice(totalInvested))
	text += fmt.Sprintf("💰 Остаток: %s₽\n", formatPrice(budget-totalInvested))
	text += fmt.Sprintf("📈 Ожидаемая прибыль: %s₽\n", formatPrice(totalExpectedProfit))
	text += fmt.Sprintf("🎯 Общий ROI: %.0f%%\n\n", expectedROI)

	text += "🛒 *Рекомендуемые покупки:*\n\n"

	for i, rec := range recommendations {
		emoji := b.getCategoryEmoji(rec.Category)
		text += fmt.Sprintf("%d. %s *%s*\n", i+1, emoji, rec.ItemName)
		text += fmt.Sprintf("   💸 %s₽ × %d шт = %s₽\n", 
			formatPrice(rec.Price), rec.Quantity, formatPrice(rec.TotalCost))
		text += fmt.Sprintf("   📈 ROI: %.0f%% (+%s₽)\n", 
			rec.ExpectedROI*100, formatPrice(rec.ExpectedProfit))
		text += fmt.Sprintf("   ⭐ Рейтинг: %.1f/10\n\n", rec.TrendScore)
	}

	text += "⚠️ *Важно:*\n"
	text += "• Это прогноз, реальная доходность может отличаться\n"
	text += "• Инвестируйте только те средства, которые готовы потерять\n"
	text += "• Рекомендуемый срок холда: 6-12 месяцев\n"
	text += "• Следите за обновлениями игры и рынка"

	// Разбиваем длинное сообщение если нужно
	if len(text) > 4000 {
		// Отправляем первую часть
		firstPart := text[:4000]
		lastNewline := strings.LastIndex(firstPart, "\n")
		if lastNewline > 0 {
			firstPart = firstPart[:lastNewline]
		}
		
		msg := tgbotapi.NewMessage(chatID, firstPart)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)
		
		// Отправляем вторую часть
		secondPart := text[len(firstPart):]
		msg = tgbotapi.NewMessage(chatID, secondPart)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		b.api.Send(msg)
	}

	// Добавляем кнопку для нового расчета
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Новый расчет", "budget_new"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Топ предметы", "back_to_top"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "💡 Хотите пересчитать с другим бюджетом?")
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// Фильтрация предметов по категории
func filterItemsByCategory(items []analyzer.ItemTrend, category string) []analyzer.ItemTrend {
	var filtered []analyzer.ItemTrend
	for _, item := range items {
		if item.Category == category {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// Форматирование цены
func formatPrice(price float64) string {
	if price >= 1000000 {
		return fmt.Sprintf("%.1fM", price/1000000)
	} else if price >= 1000 {
		return fmt.Sprintf("%.0fK", price/1000)
	}
	return fmt.Sprintf("%.0f", price)
}