package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var monthMap = map[string]string{
	"janeiro":   "01",
	"fevereiro": "02",
	"março":     "03",
	"abril":     "04",
	"maio":      "05",
	"junho":     "06",
	"julho":     "07",
	"agosto":    "08",
	"setembro":  "09",
	"outubro":   "10",
	"novembro":  "11",
	"dezembro":  "12",
}

func extractMeetingNumber(title string) int {
	re := regexp.MustCompile(`(\d+)ª`)
	matches := re.FindStringSubmatch(title)
	if len(matches) > 1 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}

func extractDateFromContent(content string) (string, error) {
	re := regexp.MustCompile(`(\d{1,2})\s+de\s+([a-zA-Zç]+)\s+de\s+(\d{4})`)
	matches := re.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return "", errors.New("nenhuma data encontrada no formato 'dia de mes de ano'")
	}

	lastMatch := matches[len(matches)-1]
	diaStr := lastMatch[1]
	mesStr := strings.ToLower(lastMatch[2])
	anoStr := lastMatch[3]

	mesNum, ok := monthMap[mesStr]
	if !ok {
		return "", fmt.Errorf("mês desconhecido: %s", mesStr)
	}

	dia, _ := strconv.Atoi(diaStr)
	diaFormatado := fmt.Sprintf("%02d", dia)

	return fmt.Sprintf("%s-%s-%s", anoStr, mesNum, diaFormatado), nil
}

var reDateFromURL = regexp.MustCompile(`.*/(\d{8})$`)

func extractDateFromURL(url string) (string, error) {
	matches := reDateFromURL.FindStringSubmatch(url)

	if len(matches) < 2 {
		return "", errors.New("não foi encontrada data no formato /DDMMYYYY no final da URL")
	}
	dateStr := matches[1]

	if len(dateStr) != 8 {
		return "", fmt.Errorf("data na URL com formato inválido: %s", dateStr)
	}

	dia := dateStr[0:2]
	mes := dateStr[2:4]
	ano := dateStr[4:8]

	return fmt.Sprintf("%s-%s-%s", ano, mes, dia), nil
}

// --- Persistência ---

func LoadAtas(filename string) ([]CopomAta, error) {
	// Se arquivo não existe, retorna vazio
	// (Poderia checar os.IsNotExist, mas vamos simplificar)
	file, err := os.Open(filename)
	if err != nil {
		return []CopomAta{}, nil
	}
	defer file.Close()

	var atas []CopomAta
	if err := json.NewDecoder(file).Decode(&atas); err != nil {
		return nil, err
	}
	return atas, nil
}

func SaveAtas(filename string, atas []CopomAta) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(atas)
}

func LoadEnrichedData(filename string) ([]EnrichedParagraph, error) {
	file, err := os.Open(filename)
	if err != nil {
		return []EnrichedParagraph{}, nil
	}
	defer file.Close()

	var data []EnrichedParagraph
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func SaveEnrichedData(filename string, data []EnrichedParagraph) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
