package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginatedResponse representa a resposta paginada
type PaginatedResponse struct {
	Data       []EnrichedParagraph `json:"data"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	Total      int                 `json:"total"`
	TotalPages int                 `json:"total_pages"`
}

// ErrorResponse representa uma resposta de erro
type ErrorResponse struct {
	Error string `json:"error"`
}

// ListAtas godoc
// @Summary Lista todas as atas do COPOM
// @Description Retorna metadados de todas as atas (sem o conteúdo completo)
// @Tags Atas
// @Produce json
// @Success 200 {array} CopomAta
// @Router /atas [get]
func ListAtas(store *ataStore) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

// GetAtaByNumero godoc
// @Summary Busca ata por número da reunião
// @Description Retorna uma ata específica com conteúdo completo
// @Tags Atas
// @Produce json
// @Param numero path int true "Número da reunião"
// @Success 200 {object} CopomAta
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /atas/{numero} [get]
func GetAtaByNumero(store *ataStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		numStr := c.Param("numero")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Número da reunião inválido."})
			return
		}
		store.mu.RLock()
		defer store.mu.RUnlock()
		ata, found := store.atasPorNumero[num]
		if !found {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("Ata número %d não encontrada.", num)})
			return
		}
		c.JSON(http.StatusOK, ata)
	}
}

// ListAtaNumeros godoc
// @Summary Lista números das reuniões disponíveis
// @Description Retorna array com os números de todas as reuniões (ordenado decrescente)
// @Tags Atas
// @Produce json
// @Success 200 {array} int
// @Router /atas/numeros [get]
func ListAtaNumeros(store *ataStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		store.mu.RLock()
		defer store.mu.RUnlock()

		numeros := make([]int, 0, len(store.atasPorNumero))
		for k := range store.atasPorNumero {
			numeros = append(numeros, k)
		}

		sort.Sort(sort.Reverse(sort.IntSlice(numeros)))
		c.JSON(http.StatusOK, numeros)
	}
}

// ListEnriched godoc
// @Summary Lista parágrafos enriquecidos (paginado)
// @Description Retorna parágrafos com análise de sentimento do Gemini AI
// @Tags Enriched
// @Produce json
// @Param page query int false "Número da página" default(1)
// @Param limit query int false "Itens por página (máx 100)" default(20)
// @Param meeting query int false "Filtrar por número da reunião"
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} ErrorResponse
// @Router /enriched [get]
func ListEnriched(enriched *enrichedStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		enriched.mu.RLock()
		defer enriched.mu.RUnlock()

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		meetingFilter := c.Query("meeting")

		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		var source []EnrichedParagraph
		if meetingFilter != "" {
			meetingNum, err := strconv.Atoi(meetingFilter)
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Parâmetro 'meeting' inválido."})
				return
			}
			source = enriched.byMeetingNumber[meetingNum]
		} else {
			source = enriched.paragraphs
		}

		total := len(source)
		start := (page - 1) * limit
		end := start + limit

		if start >= total {
			c.JSON(http.StatusOK, PaginatedResponse{
				Data:       []EnrichedParagraph{},
				Page:       page,
				Limit:      limit,
				Total:      total,
				TotalPages: (total + limit - 1) / limit,
			})
			return
		}

		if end > total {
			end = total
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Data:       source[start:end],
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		})
	}
}

// GetEnrichedByID godoc
// @Summary Busca parágrafo por ID global
// @Description Retorna um parágrafo enriquecido específico
// @Tags Enriched
// @Produce json
// @Param id path int true "Global ID do parágrafo"
// @Success 200 {object} EnrichedParagraph
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /enriched/{id} [get]
func GetEnrichedByID(enriched *enrichedStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ID inválido."})
			return
		}

		enriched.mu.RLock()
		defer enriched.mu.RUnlock()

		p, found := enriched.byGlobalID[id]
		if !found {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("Parágrafo com global_id %d não encontrado.", id)})
			return
		}

		c.JSON(http.StatusOK, p)
	}
}

// GetEnrichedByMeeting godoc
// @Summary Lista parágrafos de uma reunião específica
// @Description Retorna todos os parágrafos enriquecidos de uma reunião do COPOM
// @Tags Enriched
// @Produce json
// @Param numero path int true "Número da reunião"
// @Success 200 {array} EnrichedParagraph
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /enriched/meeting/{numero} [get]
func GetEnrichedByMeeting(enriched *enrichedStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		numStr := c.Param("numero")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Número da reunião inválido."})
			return
		}

		enriched.mu.RLock()
		defer enriched.mu.RUnlock()

		paragraphs, found := enriched.byMeetingNumber[num]
		if !found || len(paragraphs) == 0 {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("Nenhum parágrafo enriquecido para a reunião %d.", num)})
			return
		}

		c.JSON(http.StatusOK, paragraphs)
	}
}
