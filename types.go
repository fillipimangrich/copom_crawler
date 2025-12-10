package main

import "sync"

type CopomAta struct {
	NumeroReuniao int     `json:"numero_reuniao"`
	URL           string  `json:"url"`
	Titulo        string  `json:"titulo"`
	DataReuniao   string  `json:"data_reuniao,omitempty"` // Formato YYYY-MM-DD
	ValorDolar    float64 `json:"valor_dolar,omitempty"`  // Dólar PTAX na data
	ValorIPCA     float64 `json:"valor_ipca,omitempty"`   // IPCA do mês da reunião
	Conteudo      string  `json:"conteudo,omitempty"`
	FalhaNoParse  bool    `json:"falha_no_parse,omitempty"`
}

type GeminiPrediction struct {
	DollarTrend string `json:"dollar_trend"` // "SUBIR", "DESCER", "NEUTRO"
	IPCATrend   string `json:"ipca_trend"`   // "SUBIR", "DESCER", "NEUTRO"
	Reasoning   string `json:"reasoning"`
}

type EnrichedParagraph struct {
	MeetingNumber int              `json:"meeting_number"`
	MeetingDate   string           `json:"meeting_date"`
	DollarValue   float64          `json:"dollar_value"`
	IPCAValue     float64          `json:"ipca_value"`
	Paragraph     string           `json:"paragraph"`
	Prediction    GeminiPrediction `json:"prediction"`
}

type ataStore struct {
	mu            sync.RWMutex
	atas          []CopomAta
	atasPorNumero map[int]CopomAta
}

func newAtaStore() *ataStore {
	return &ataStore{
		atas:          make([]CopomAta, 0),
		atasPorNumero: make(map[int]CopomAta),
	}
}
