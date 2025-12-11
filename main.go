// @title COPOM Crawler API
// @version 1.0
// @description API para acesso às atas do COPOM e dados enriquecidos com análise de sentimento via Gemini AI
// @host localhost:8080
// @BasePath /

package main

import (
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/seu-usuario/copom-crawler/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

	// Backfill de IDs se necessário (para compatibilidade com dados antigos)
	needsSave := false
	maxGlobalID := 0
	paraCount := make(map[int]int)

	// Primeiro passo: identificar IDs existentes para não sobrescrever incorretamente
	for _, item := range enrichedData {
		if item.GlobalID > maxGlobalID {
			maxGlobalID = item.GlobalID
		}
		if item.ParagraphID > paraCount[item.MeetingNumber] {
			paraCount[item.MeetingNumber] = item.ParagraphID
		}
	}

	// Criar mapa de URLs das atas brutas para backfill
	meetingURLMap := make(map[int]string)
	for _, ata := range rawAtas {
		meetingURLMap[ata.NumeroReuniao] = ata.URL
	}

	// Segundo passo: preencher zeros e URLs faltantes
	for i := range enrichedData {
		if enrichedData[i].GlobalID == 0 {
			maxGlobalID++
			enrichedData[i].GlobalID = maxGlobalID
			needsSave = true
		}
		if enrichedData[i].ParagraphID == 0 {
			paraCount[enrichedData[i].MeetingNumber]++
			enrichedData[i].ParagraphID = paraCount[enrichedData[i].MeetingNumber]
			needsSave = true
		}
		if enrichedData[i].URL == "" {
			if url, ok := meetingURLMap[enrichedData[i].MeetingNumber]; ok {
				enrichedData[i].URL = url
				needsSave = true
			}
		}
	}

	if needsSave {
		log.Println("Atualizando dataset com IDs sequenciais (Backfill)...")
		if err := SaveEnrichedData(enrichedFilename, enrichedData); err != nil {
			log.Printf("Erro ao salvar backfill: %v", err)
		}
	}

	// Mapa para rastrear parágrafos já processados: MeetingNumber -> ParagraphID -> bool
	processedMap := make(map[int]map[int]bool)
	nextGlobalID := 1

	// Inicializar mapa e encontrar o próximo GlobalID
	for _, item := range enrichedData {
		if _, ok := processedMap[item.MeetingNumber]; !ok {
			processedMap[item.MeetingNumber] = make(map[int]bool)
		}
		processedMap[item.MeetingNumber][item.ParagraphID] = true

		if item.GlobalID >= nextGlobalID {
			nextGlobalID = item.GlobalID + 1
		}
	}

	log.Printf("Total de atas brutas: %d", len(rawAtas))
	log.Printf("Total de parágrafos já enriquecidos: %d", len(enrichedData))
	log.Printf("Próximo Global ID: %d", nextGlobalID)

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
		// Não pulamos a ata inteira aqui baseada no enrichedMap antigo, pois precisamos checar parágrafo a parágrafo.
		// Mas se já processamos TODOS os parágrafos dessa ata, poderíamos pular.
		// Por simplificação, vamos gerar os parágrafos e checar um a um.

		if maxMeetings > 0 && count >= maxMeetings {
			log.Printf("Atingido limite de processamento de atas (%d). Parando.", maxMeetings)
			break
		}

		// Limite de segurança total de parágrafos (ex: 1000)
		if len(enrichedData) >= 1000 {
			log.Printf("Atingido limite total de 1000 parágrafos enriquecidos. Parando.")
			break
		}

		if ata.Conteudo == "" {
			continue
		}

		log.Printf("Analisando Ata %d...", ata.NumeroReuniao)

		var textContent string
		if ata.FalhaNoParse {
			// Remover tags HTML simples para extrair texto
			re := regexp.MustCompile(`<[^>]*>`)
			textContent = re.ReplaceAllString(ata.Conteudo, "\n")
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
			if currentBuffer.Len() > 0 {
				currentBuffer.WriteString(" ")
			}
			currentBuffer.WriteString(line)
			if currentBuffer.Len() >= 200 {
				paragraphs = append(paragraphs, currentBuffer.String())
				currentBuffer.Reset()
			}
		}
		if currentBuffer.Len() > 0 {
			paragraphs = append(paragraphs, currentBuffer.String())
		}

		processedParagraphs := 0
		newlyEnrichedCount := 0
		totalParagraphs := len(paragraphs)

		// Inicializar mapa para esta ata se não existir
		if _, ok := processedMap[ata.NumeroReuniao]; !ok {
			processedMap[ata.NumeroReuniao] = make(map[int]bool)
		}

		for i, p := range paragraphs {
			paragraphID := i + 1 // ID sequencial base 1

			// Verificação extra de tamanho mínimo
			if len(p) < 50 {
				continue
			}

			// Checar se já foi processado
			if processedMap[ata.NumeroReuniao][paragraphID] {
				continue
			}

			if (i+1)%5 == 0 || i == 0 {
				log.Printf("  Processando parágrafo %d/%d da Ata %d...", paragraphID, totalParagraphs, ata.NumeroReuniao)
			}

			prediction, err := callGeminiAPI(p, ata.ValorDolar, ata.ValorIPCA)
			if err != nil {
				log.Printf("Erro ao chamar Gemini para reunião %d: %v", ata.NumeroReuniao, err)
				time.Sleep(5 * time.Second)
				continue
			}

			enriched := EnrichedParagraph{
				GlobalID:      nextGlobalID,
				ParagraphID:   paragraphID,
				MeetingNumber: ata.NumeroReuniao,
				URL:           ata.URL,
				MeetingDate:   ata.DataReuniao,
				DollarValue:   ata.ValorDolar,
				IPCAValue:     ata.ValorIPCA,
				Paragraph:     p,
				Prediction:    prediction,
			}
			enrichedData = append(enrichedData, enriched)
			processedMap[ata.NumeroReuniao][paragraphID] = true
			nextGlobalID++
			processedParagraphs++
			newlyEnrichedCount++

			// Rate limit
			time.Sleep(2 * time.Second)
		}

		if newlyEnrichedCount > 0 {
			if err := SaveEnrichedData(enrichedFilename, enrichedData); err != nil {
				log.Printf("Erro ao salvar dados enriquecidos: %v", err)
			} else {
				log.Printf("Ata %d salva com %d novos parágrafos enriquecidos.", ata.NumeroReuniao, newlyEnrichedCount)
				count++
			}
		} else {
			log.Printf("Ata %d: nenhum novo parágrafo para enriquecer.", ata.NumeroReuniao)
		}
	}
	log.Println("Enrichment finalizado.")
}

func runServer() {
	log.Println("=== MODO SERVER ===")

	// Carregar dados das atas
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

	// Carregar dados enriquecidos
	enriched := newEnrichedStore()
	enrichedData, err := LoadEnrichedData("dataset_enriched.json")
	if err != nil {
		log.Printf("AVISO: Não foi possível carregar dataset_enriched.json: %v", err)
	}

	enriched.mu.Lock()
	enriched.paragraphs = enrichedData
	for _, p := range enrichedData {
		enriched.byGlobalID[p.GlobalID] = p
		enriched.byMeetingNumber[p.MeetingNumber] = append(enriched.byMeetingNumber[p.MeetingNumber], p)
	}
	enriched.mu.Unlock()
	log.Printf("Servindo %d parágrafos enriquecidos.", len(enriched.paragraphs))

	// Configurar router
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Endpoints de Atas
	router.GET("/atas", ListAtas(store))
	router.GET("/atas/numeros", ListAtaNumeros(store))
	router.GET("/atas/:numero", GetAtaByNumero(store))

	// Endpoints de dados enriquecidos
	router.GET("/enriched", ListEnriched(enriched))
	router.GET("/enriched/:id", GetEnrichedByID(enriched))
	router.GET("/enriched/meeting/:numero", GetEnrichedByMeeting(enriched))

	log.Println("Servidor de API iniciado em http://localhost:8080")
	router.Run(":8080")
}
