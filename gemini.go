package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiModel = "gemini-2.5-flash-lite"
const geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models/" + geminiModel + ":generateContent"

// Structs para requisição e resposta da API do Gemini
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func callGeminiAPI(paragraph string, dollar float64, ipca float64) (GeminiPrediction, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return GeminiPrediction{}, fmt.Errorf("GEMINI_API_KEY não definida")
	}

	prompt := fmt.Sprintf(`
Analise o seguinte parágrafo da Ata do COPOM e os dados econômicos fornecidos.
Faça uma predição de tendência para o Dólar e para o IPCA (inflação) com base no tom e conteúdo do texto.

Dados:
- Dólar PTAX (dia anterior à reunião): %.4f
- IPCA (mês da reunião): %.2f%%
- Parágrafo da Ata: "%s"

Responda APENAS com um JSON no seguinte formato, sem markdown ou explicações adicionais:
{
  "dollar_trend": "SUBIR" | "DESCER" | "NEUTRO",
  "ipca_trend": "SUBIR" | "DESCER" | "NEUTRO",
  "reasoning": "Breve explicação do porquê (máx 1 frase)"
}
`, dollar, ipca, paragraph)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return GeminiPrediction{}, err
	}

	url := fmt.Sprintf("%s?key=%s", geminiAPIURL, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return GeminiPrediction{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return GeminiPrediction{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return GeminiPrediction{}, fmt.Errorf("erro na API Gemini (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return GeminiPrediction{}, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return GeminiPrediction{}, fmt.Errorf("resposta vazia do Gemini")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Limpar markdown se houver (```json ... ```)
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
	}
	responseText = strings.TrimSpace(responseText)

	var prediction GeminiPrediction
	if err := json.Unmarshal([]byte(responseText), &prediction); err != nil {
		log.Printf("Erro ao parsear JSON do Gemini: %s", responseText)
		return GeminiPrediction{Reasoning: "Erro no parse da resposta"}, nil // Retorna vazio mas não erro fatal
	}

	return prediction, nil
}
