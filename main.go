package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func getAtaNumbers(c *gin.Context, store *ataStore) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	numeros := make([]int, 0, len(store.atasPorNumero))
	for k := range store.atasPorNumero {
		numeros = append(numeros, k)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(numeros)))

	c.JSON(http.StatusOK, numeros)
}

func main() {
	modePtr := flag.String("mode", "serve", "Modo de operação: 'scrape', 'enrich', 'serve' ou 'all'")
	flag.Parse()

	switch *modePtr {
	case "scrape":
		runScraper()
	case "enrich":
		runEnricher()
	case "serve":
		runServer()
	case "all":
		runScraper()
		runEnricher()
		runServer()
	default:
		log.Fatalf("Modo desconhecido: %s. Use -mode=scrape, -mode=enrich, -mode=serve ou -mode=all", *modePtr)
	}
}

func runScraper() {
	log.Println("=== MODO SCRAPER ===")
	filename := "dataset_raw.json"

	// Carregar dados existentes
	existingAtas, err := LoadAtas(filename)
	if err != nil {
		log.Printf("Erro ao carregar %s (será criado novo): %v", filename, err)
	}

	existingMap := make(map[int]bool)
	for _, ata := range existingAtas {
		if ata.NumeroReuniao != 0 && !ata.FalhaNoParse {
			existingMap[ata.NumeroReuniao] = true
		}
	}
	log.Printf("Carregadas %d atas existentes.", len(existingAtas))

	// Callback para salvar a cada nova ata
	onSave := func(newAta CopomAta) error {
		// Adicionar ou atualizar na lista em memória
		found := false
		for i, ata := range existingAtas {
			if ata.NumeroReuniao == newAta.NumeroReuniao {
				existingAtas[i] = newAta
				found = true
				break
			}
		}
		if !found {
			existingAtas = append(existingAtas, newAta)
		}
		// Salvar tudo no disco
		// (Ineficiente para grandes volumes, mas seguro e simples para <100MB)
		return SaveAtas(filename, existingAtas)
	}

	if err := scrapeCopomAtas(existingMap, onSave); err != nil {
		log.Printf("Erro durante o scraping: %v", err)
	}
	log.Println("Scraping finalizado.")
}

func runEnricher() {
	log.Println("=== MODO ENRICHER (GEMINI) ===")
	rawFilename := "dataset_raw.json"
	enrichedFilename := "dataset_enriched.json"

	rawAtas, err := LoadAtas(rawFilename)
	if err != nil {
		log.Fatalf("Erro ao carregar %s: %v. Execute o modo 'scrape' primeiro.", rawFilename, err)
	}

	enrichedData, err := LoadEnrichedData(enrichedFilename)
	if err != nil {
		log.Printf("Erro ao carregar %s (será criado novo): %v", enrichedFilename, err)
	}

	enrichedMap := make(map[int]bool)
	for _, item := range enrichedData {
		enrichedMap[item.MeetingNumber] = true
	}

	log.Printf("Total de atas brutas: %d", len(rawAtas))
	log.Printf("Total de atas já enriquecidas: %d", len(enrichedData))

	// Configuração de limite (opcional)
	maxMeetingsStr := os.Getenv("MAX_MEETINGS")
	maxMeetings := 0 // 0 = sem limite
	if maxMeetingsStr != "" {
		if val, err := strconv.Atoi(maxMeetingsStr); err == nil {
			maxMeetings = val
		}
	}

	count := 0
	for _, ata := range rawAtas {
		if enrichedMap[ata.NumeroReuniao] {
			continue
		}

		if maxMeetings > 0 && count >= maxMeetings {
			log.Printf("Atingido limite de processamento (%d). Parando.", maxMeetings)
			break
		}

		if ata.Conteudo == "" {
			continue
		}

		log.Printf("Enriquecendo Ata %d...", ata.NumeroReuniao)

		var textContent string
		if ata.FalhaNoParse {
			// Remover tags HTML simples para extrair texto
			// Regex simples para remover tags
			re := regexp.MustCompile(`<[^>]*>`)
			textContent = re.ReplaceAllString(ata.Conteudo, "\n")
			// Decodificar entidades HTML básicas se necessário (simplificado aqui)
			textContent = strings.ReplaceAll(textContent, "&nbsp;", " ")
			textContent = strings.ReplaceAll(textContent, "&amp;", "&")
		} else {
			textContent = ata.Conteudo
		}

		// Quebrar em linhas e agregar parágrafos
		rawLines := strings.Split(textContent, "\n")
		var paragraphs []string
		var currentBuffer strings.Builder

		for _, line := range rawLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Se o buffer não estiver vazio, adicionar espaço antes da nova linha
			if currentBuffer.Len() > 0 {
				currentBuffer.WriteString(" ")
			}
			currentBuffer.WriteString(line)

			// Se o parágrafo acumulado for grande o suficiente, considerar um parágrafo
			// Aumentei para 200 chars para evitar frases soltas, mas pode ajustar
			if currentBuffer.Len() >= 200 {
				paragraphs = append(paragraphs, currentBuffer.String())
				currentBuffer.Reset()
			}
		}
		// Adicionar o restante do buffer se houver algo
		if currentBuffer.Len() > 0 {
			paragraphs = append(paragraphs, currentBuffer.String())
		}

		processedParagraphs := 0
		for _, p := range paragraphs {
			// Verificação extra de tamanho mínimo (embora a agregação já ajude)
			if len(p) < 50 {
				continue
			}

			prediction, err := callGeminiAPI(p, ata.ValorDolar, ata.ValorIPCA)
			if err != nil {
				log.Printf("Erro ao chamar Gemini para reunião %d: %v", ata.NumeroReuniao, err)
				// Rate limit backoff
				time.Sleep(5 * time.Second)
				continue
			}

			enriched := EnrichedParagraph{
				MeetingNumber: ata.NumeroReuniao,
				MeetingDate:   ata.DataReuniao,
				DollarValue:   ata.ValorDolar,
				IPCAValue:     ata.ValorIPCA,
				Paragraph:     p,
				Prediction:    prediction,
			}
			enrichedData = append(enrichedData, enriched)
			processedParagraphs++

			// Rate limit
			time.Sleep(2 * time.Second)
		}

		if processedParagraphs > 0 {
			// Salvar progresso após cada ata processada
			if err := SaveEnrichedData(enrichedFilename, enrichedData); err != nil {
				log.Printf("Erro ao salvar dados enriquecidos: %v", err)
			} else {
				log.Printf("Ata %d salva com %d parágrafos enriquecidos.", ata.NumeroReuniao, processedParagraphs)
				enrichedMap[ata.NumeroReuniao] = true
				count++
			}
		} else {
			log.Printf("Ata %d não gerou parágrafos válidos.", ata.NumeroReuniao)
		}
	}
	log.Println("Enrichment finalizado.")
}

func runServer() {
	log.Println("=== MODO SERVER ===")
	// Carregar dados para servir
	// A API original servia 'store.atas' (CopomAta).
	// Vamos carregar dataset_raw.json para manter compatibilidade.

	store := newAtaStore()
	atas, err := LoadAtas("dataset_raw.json")
	if err != nil {
		log.Printf("AVISO: Não foi possível carregar dataset_raw.json: %v", err)
	}

	store.mu.Lock()
	store.atas = atas
	for _, ata := range atas {
		if ata.NumeroReuniao != 0 {
			store.atasPorNumero[ata.NumeroReuniao] = ata
		}
	}
	store.mu.Unlock()

	log.Printf("Servindo %d atas.", len(store.atas))

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/atas", func(c *gin.Context) {
		store.mu.RLock()
		defer store.mu.RUnlock()
		var atasSemConteudo []CopomAta
		for _, ata := range store.atas {
			atasSemConteudo = append(atasSemConteudo, CopomAta{
				NumeroReuniao: ata.NumeroReuniao,
				URL:           ata.URL,
				Titulo:        ata.Titulo,
				DataReuniao:   ata.DataReuniao,
				ValorDolar:    ata.ValorDolar,
				ValorIPCA:     ata.ValorIPCA,
			})
		}
		c.JSON(http.StatusOK, atasSemConteudo)
	})

	router.GET("/atas/:numero", func(c *gin.Context) {
		numStr := c.Param("numero")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Número da reunião inválido."})
			return
		}
		store.mu.RLock()
		defer store.mu.RUnlock()
		ata, found := store.atasPorNumero[num]
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Ata número %d não encontrada.", num)})
			return
		}
		c.JSON(http.StatusOK, ata)
	})

	router.GET("/atas/numeros", func(c *gin.Context) {
		getAtaNumbers(c, store)
	})

	log.Println("Servidor de API iniciado em http://localhost:8080")
	router.Run(":8080")
}
