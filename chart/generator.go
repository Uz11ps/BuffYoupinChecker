package chart

import (
	"bytes"
	"fmt"
	"time"

	"buff-youpin-checker/database"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

type ChartGenerator struct {
	db *database.DB
}

func NewChartGenerator(db *database.DB) *ChartGenerator {
	return &ChartGenerator{db: db}
}

func (cg *ChartGenerator) GeneratePriceChart(itemID int, days int) ([]byte, error) {
	// Получаем историю цен
	query := `SELECT price, recorded_at FROM price_history 
			  WHERE item_id = $1 AND recorded_at >= $2 
			  ORDER BY recorded_at ASC`
	
	startDate := time.Now().AddDate(0, 0, -days)
	rows, err := cg.db.Query(query, itemID, startDate)
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

	if len(prices) == 0 {
		return nil, fmt.Errorf("no price data found")
	}

	// Создаем график
	graph := chart.Chart{
		Title: "Динамика цены",
		TitleStyle: chart.Style{
			FontSize: 16,
		},
		Width:  800,
		Height: 400,
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   20,
				Right:  20,
				Bottom: 20,
			},
		},
		XAxis: chart.XAxis{
			Name: "Дата",
			Style: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		YAxis: chart.YAxis{
			Name: "Цена (₽)",
		},
		Series: []chart.Series{
			chart.TimeSeries{
				Name: "Цена",
				Style: chart.Style{
					StrokeColor: drawing.ColorBlue,
					StrokeWidth: 2,
				},
				XValues: timestamps,
				YValues: prices,
			},
		},
	}

	// Добавляем линию тренда если данных достаточно
	if len(prices) > 5 {
		trendSeries := cg.calculateTrendLine(timestamps, prices)
		graph.Series = append(graph.Series, trendSeries)
	}

	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	// Рендерим в буфер
	buffer := bytes.NewBuffer([]byte{})
	err = graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (cg *ChartGenerator) calculateTrendLine(timestamps []time.Time, prices []float64) chart.TimeSeries {
	if len(prices) < 2 {
		return chart.TimeSeries{}
	}

	// Простая линейная регрессия
	n := float64(len(prices))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumX2 += x * x
	}

	// Коэффициенты линейной регрессии
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Создаем точки для линии тренда
	var trendPrices []float64
	for i := range prices {
		trendPrice := intercept + slope*float64(i)
		trendPrices = append(trendPrices, trendPrice)
	}

	return chart.TimeSeries{
		Name: "Тренд",
		Style: chart.Style{
			StrokeColor:     drawing.ColorRed,
			StrokeWidth:     2,
			StrokeDashArray: []float64{5.0, 5.0},
		},
		XValues: timestamps,
		YValues: trendPrices,
	}
}