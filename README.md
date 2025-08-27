# 🎯 BuffYoupinChecker

<div align="center">

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/postgresql-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)

**Умный Telegram-бот для анализа цен на скины CS:GO**

*Отслеживайте тренды рынка, получайте аналитику и принимайте обоснованные решения*

</div>

---

## 📋 Описание

BuffYoupinChecker — это мощный инструмент для анализа рынка скинов CS:GO, который:

- 📊 **Собирает данные** с market.csgo.com в реальном времени
- 📈 **Анализирует тренды** и волатильность цен
- 🤖 **Предоставляет рекомендации** через Telegram-бота
- 📉 **Строит графики** динамики цен
- 🔍 **Отслеживает** перспективные предметы

## ✨ Возможности

### 🤖 Telegram Бот
- Поиск предметов по названию
- Просмотр текущих цен и трендов
- Получение рекомендаций по покупке/продаже
- Генерация графиков цен
- Топ растущих и падающих предметов

### 📊 Аналитика
- Расчет темпов роста цен
- Анализ волатильности
- Система оценки трендов (1-10)
- Автоматические рекомендации
- Исторические данные

### 🔄 Автоматизация
- Периодический сбор данных (каждые 10 минут)
- Автоматический анализ трендов (каждые 30 минут)
- Уведомления о значительных изменениях

## 🚀 Быстрый старт

### Предварительные требования

- Go 1.21 или выше
- PostgreSQL 12+
- Telegram Bot Token
- API ключ от market.csgo.com

### 📦 Установка

1. **Клонируйте репозиторий:**
```bash
git clone https://github.com/Uz11ps/BuffYoupinChecker.git
cd BuffYoupinChecker
```

2. **Установите зависимости:**
```bash
go mod download
```

3. **Настройте базу данных:**
```bash
# Создайте базу данных PostgreSQL
createdb skin_analyzer

# Выполните миграции
psql -d skin_analyzer -f database/schema.sql
```

4. **Настройте переменные окружения:**
```bash
cp .env.example .env
# Отредактируйте .env файл с вашими настройками
```

5. **Запустите приложение:**
```bash
go run main.go
```

## ⚙️ Конфигурация

Создайте файл `.env` в корне проекта:

```env
# Telegram Bot
TELEGRAM_BOT_TOKEN=your_telegram_bot_token

# Market API
MARKET_API_KEY=your_market_api_key

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=skin_analyzer

# Server
PORT=8080
```

## 🎮 Использование

### Команды бота

- `/start` - Начать работу с ботом
- `/help` - Справка по командам
- `/search <название>` - Поиск предмета
- `/top_growing` - Топ растущих предметов
- `/top_falling` - Топ падающих предметов
- `/trends` - Общие тренды рынка

### Примеры использования

```
/search AK-47 Redline
/top_growing 10
/trends
```

### Интерпретация результатов

- **Рейтинг 8-10**: 🟢 Отличные перспективы (BUY)
- **Рейтинг 6-7**: 🟡 Умеренные перспективы (HOLD)  
- **Рейтинг 1-5**: 🔴 Низкие перспективы (SELL)

## 🏗️ Архитектура

```
BuffYoupinChecker/
├── analyzer/          # Модуль анализа трендов
├── bot/              # Telegram бот
├── chart/            # Генерация графиков
├── config/           # Конфигурация
├── database/         # Работа с БД
├── market/           # API клиент market.csgo.com
└── main.go           # Точка входа
```

### Как работает анализ

1. **Сбор данных**: Каждые 10 минут собираются актуальные цены с market.csgo.com
2. **Анализ трендов**: Каждые 30 минут анализируются ценовые тренды
3. **Рейтинг**: Скины оцениваются от 1 до 10 на основе:
   - Процента роста цены
   - Волатильности
   - Стабильности тренда
   - Объема данных

## 🛠️ Разработка

### Сборка

```bash
# Сборка для текущей платформы
go build -o buff-youpin-checker

# Кросс-компиляция для Windows
GOOS=windows GOARCH=amd64 go build -o buff-youpin-checker.exe
```

### Тестирование

```bash
go test ./...
```

## ⚠️ Ограничения API

**Важно**: market.csgo.com ограничивает до 5 запросов в секунду. Превышение лимита приведет к блокировке API ключа.

Бот использует rate limiting (4 запроса/сек) для безопасности.

## 🚀 Будущие возможности

- [ ] Интеграция с Buff163, Youpin, Lis
- [ ] Обменник рубль/юань  
- [ ] Автоматический выкуп скинов
- [ ] Уведомления о выгодных предложениях
- [ ] Веб-интерфейс для аналитики
- [ ] REST API для внешних интеграций

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции (`git checkout -b feature/amazing-feature`)
3. Зафиксируйте изменения (`git commit -m 'Add amazing feature'`)
4. Отправьте в ветку (`git push origin feature/amazing-feature`)
5. Откройте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

## 📞 Поддержка

- 🐛 **Баги**: [Issues](https://github.com/Uz11ps/BuffYoupinChecker/issues)
- 💡 **Предложения**: [Discussions](https://github.com/Uz11ps/BuffYoupinChecker/discussions)
- 📧 **Email**: support@example.com

---

<div align="center">

**⭐ Поставьте звезду, если проект был полезен!**

Made with ❤️ by [Uz11ps](https://github.com/Uz11ps)

</div>