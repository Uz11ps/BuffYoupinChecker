-- Создание базы данных
CREATE DATABASE IF NOT EXISTS skin_analyzer;

-- Таблица предметов
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    hash_name VARCHAR(255) UNIQUE NOT NULL,
    market_name VARCHAR(255) NOT NULL,
    class_id VARCHAR(50),
    instance_id VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица цен (история)
CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    item_id INTEGER REFERENCES items(id),
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'RUB',
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(50) DEFAULT 'market.csgo.com'
);

-- Таблица анализа предметов
CREATE TABLE IF NOT EXISTS item_analysis (
    id SERIAL PRIMARY KEY,
    item_id INTEGER REFERENCES items(id),
    growth_rate DECIMAL(5,2), -- процент роста
    volatility DECIMAL(5,2),  -- волатильность
    trend_score INTEGER,      -- рейтинг от 1 до 10
    recommendation VARCHAR(20), -- BUY/HOLD/SELL
    analysis_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для оптимизации
CREATE INDEX idx_items_hash_name ON items(hash_name);
CREATE INDEX idx_price_history_item_id ON price_history(item_id);
CREATE INDEX idx_price_history_recorded_at ON price_history(recorded_at);
CREATE INDEX idx_item_analysis_trend_score ON item_analysis(trend_score DESC);