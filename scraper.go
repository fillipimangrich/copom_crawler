package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

const copomListURL = "https://www.bcb.gov.br/publicacoes/atascopom/cronologicos"
const investingURL = "https://br.investing.com/currencies/usd-brl-historical-data"
const seleniumPort = 9515

func getDolarPorScraping(wd selenium.WebDriver, dataYMD string) (float64, error) {
	log.Printf("Iniciando scraping do Dólar para a data: %s", dataYMD)

	t, err := time.Parse("2006-01-02", dataYMD)
	if err != nil {
		return 0, fmt.Errorf("formato de data inválido: %s", dataYMD)
	}

	// Usar o dia anterior à reunião
	diaAnterior := t.AddDate(0, 0, -1)
	// Definir intervalo de 2 dias: Dia da reunião e dia anterior
	// O usuário sugeriu: "data anterior a da ata e a da ata"
	dataInicio := diaAnterior.Format("2006-01-02")
	dataFim := t.Format("2006-01-02")

	log.Printf("[Scraping Dólar] Buscando dados entre %s e %s", dataInicio, dataFim)

	log.Println("[Scraping Dólar] 1. Navegando para a URL do Investing...")
	if err := wd.Get(investingURL); err != nil {
		return 0, fmt.Errorf("falha ao abrir a URL do Investing: %v", err)
	}
	log.Println("[Scraping Dólar] 1. Navegação concluída.")

	log.Println("[Scraping Dólar] 2. Aguardando 2s para banner de cookies...")
	time.Sleep(2 * time.Second)
	log.Println("[Scraping Dólar] 2. Tentando clicar no banner de cookies...")
	tryToClick(wd, selenium.ByID, "onetrust-accept-btn-handler")
	log.Println("[Scraping Dólar] 2. Tentativa de clique no banner concluída.")

	log.Println("[Scraping Dólar] 3. Tentando clicar no pop-up de login...")
	tryToClick(wd, selenium.ByClassName, "popupCloseIcon")
	log.Println("[Scraping Dólar] 3. Tentativa de clique no pop-up concluída.")

	waitTimeout := 20 * time.Second

	log.Println("[Scraping Dólar] 4. Aguardando o seletor de data estar clicável...")

	datePickerXPath := "//div[contains(text(), ' - ')]/parent::div[contains(@class, 'rounded')]"
	if err := wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		el, err := wd.FindElement(selenium.ByXPATH, datePickerXPath)
		if err != nil {
			return false, nil
		}
		displayed, err := el.IsDisplayed()
		if err != nil || !displayed {
			return false, nil
		}
		enabled, err := el.IsEnabled()
		if err != nil || !enabled {
			return false, nil
		}
		return true, nil
	}, waitTimeout); err != nil {
		return 0, fmt.Errorf("botão de data (%s) não ficou clicável: %v", datePickerXPath, err)
	}
	log.Println("[Scraping Dólar] 4. Seletor de data está clicável.")
	datePickerButton, err := wd.FindElement(selenium.ByXPATH, datePickerXPath)
	if err != nil {
		return 0, fmt.Errorf("não foi possível encontrar o botão de data %s: %v", datePickerXPath, err)
	}

	log.Println("[Scraping Dólar] 5. Clicando no botão de data (via JavaScript)...")

	if _, err := wd.ExecuteScript("arguments[0].click();", []interface{}{datePickerButton}); err != nil {
		return 0, fmt.Errorf("falha ao clicar no botão de data via JS: %v", err)
	}
	log.Println("[Scraping Dólar] 5. Clique no botão de data concluído.")
	time.Sleep(1 * time.Second)

	log.Println("[Scraping Dólar] 6. Aguardando o pop-up de data abrir...")

	startDateXPath := "//div[contains(@class, 'NativeDateInputV2_root')][1]//input"
	if err := wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		el, err := wd.FindElement(selenium.ByXPATH, startDateXPath)
		if err != nil {
			return false, nil
		}
		return el != nil, nil
	}, waitTimeout); err != nil {

		screenshot, ssErr := wd.Screenshot()
		if ssErr == nil {
			filename := fmt.Sprintf("debug_screenshot_%s.png", time.Now().Format("150405"))
			os.WriteFile(filename, screenshot, 0644)
			log.Printf("[Scraping Dólar] DEBUG: Screenshot salvo em %s", filename)
		} else {
			log.Printf("[Scraping Dólar] DEBUG: Falha ao tirar screenshot: %v", ssErr)
		}

		pageSource, psErr := wd.PageSource()
		if psErr == nil {
			filename := fmt.Sprintf("debug_page_source_%s.html", time.Now().Format("150405"))
			os.WriteFile(filename, []byte(pageSource), 0644)
			log.Printf("[Scraping Dólar] DEBUG: HTML da página salvo em %s", filename)
		} else {
			log.Printf("[Scraping Dólar] DEBUG: Falha ao salvar HTML da página: %v", psErr)
		}

		return 0, fmt.Errorf("input de data (startDateXPath) não apareceu: %v", err)
	}
	log.Println("[Scraping Dólar] 6. Pop-up de data aberto (input de data encontrado).")

	// Estratégia Híbrida: Usar fetch() via JavaScript para buscar dados da API interna
	// Isso evita a interação com o DatePicker e usa a sessão do navegador para passar pelo Cloudflare
	log.Println("[Scraping Dólar] 7. Buscando dados via API interna (fetch)...")

	script := `
		var done = arguments[arguments.length - 1];
		var startDate = arguments[0];
		var endDate = arguments[1];
		var url = 'https://api.investing.com/api/financialdata/historical/2103?start-date=' + startDate + '&end-date=' + endDate + '&time-frame=Daily&add-missing-rows=false';
		
		fetch(url, {
			headers: {
				'domain-id': 'br',
				'accept': '*/*',
				'x-requested-with': 'XMLHttpRequest',
			}
		})
		.then(response => {
			if (!response.ok) {
				throw new Error('Network response was not ok: ' + response.statusText);
			}
			return response.json();
		})
		.then(data => done(JSON.stringify(data)))
		.catch(error => done('ERROR: ' + error.message));
	`

	// ExecuteScriptAsync é necessário para operações assíncronas como fetch
	result, err := wd.ExecuteScriptAsync(script, []interface{}{dataInicio, dataFim})
	if err != nil {
		return 0, fmt.Errorf("erro ao executar fetch via JS: %v", err)
	}

	jsonStr, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("resposta do fetch não é string")
	}

	if strings.HasPrefix(jsonStr, "ERROR:") {
		return 0, fmt.Errorf("erro no fetch JS: %s", jsonStr)
	}

	// Como não sabemos a estrutura exata, vamos tentar um map genérico primeiro para não quebrar
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		return 0, fmt.Errorf("erro ao fazer parse do JSON: %v", err)
	}

	// Tentar extrair os dados. Geralmente vem em "data"
	dataList, ok := rawData["data"].([]interface{})
	if !ok {
		return 0, fmt.Errorf("campo 'data' não encontrado ou inválido no JSON")
	}

	if len(dataList) == 0 {
		return 0, fmt.Errorf("nenhum dado encontrado para o período")
	}

	// Inspecionar o primeiro item
	firstItem := dataList[0].(map[string]interface{})

	// Tentar extrair o preço (last_close, price, close, etc)
	var price float64

	// Preferir o valor Raw se existir (geralmente é numérico)
	if val, ok := firstItem["last_closeRaw"]; ok {
		if f, ok := val.(float64); ok {
			price = f
		} else {
			log.Printf("[Scraping Dólar] last_closeRaw não é float64: %T", val)
		}
	}

	// Se não conseguiu, tentar last_close (string)
	if price == 0 {
		if val, ok := firstItem["last_close"]; ok {
			if s, ok := val.(string); ok {
				s = strings.Replace(s, ",", ".", 1)
				if f, err := strconv.ParseFloat(s, 64); err == nil {
					price = f
				}
			}
		}
	}

	if price == 0 {
		// Tentar 'price' como fallback
		if val, ok := firstItem["price"]; ok {
			if f, ok := val.(float64); ok {
				price = f
			}
		}
	}

	if price == 0 {
		return 0, fmt.Errorf("campo de preço não identificado ou inválido no JSON")
	}

	log.Printf("[Scraping Dólar] Preço extraído: %.4f", price)
	return price, nil
}

func getIPCAPorScraping(wd selenium.WebDriver) (map[string]float64, error) {
	log.Println("[Scraping IPCA] Iniciando extração do IPCA do IBGE...")
	url := "https://www.ibge.gov.br/estatisticas/economicas/precos-e-custos/9256-indice-nacional-de-precos-ao-consumidor-amplo.html?=&t=series-historicas"

	if err := wd.Get(url); err != nil {
		return nil, fmt.Errorf("falha ao abrir URL do IBGE: %v", err)
	}

	// Esperar o gráfico carregar (Highcharts)
	timeout := 20 * time.Second
	err := wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		res, err := wd.ExecuteScript("return (window.Highcharts && window.Highcharts.charts && window.Highcharts.charts[0]) ? true : false", nil)
		if err != nil {
			return false, nil
		}
		return res.(bool), nil
	}, timeout)

	if err != nil {
		return nil, fmt.Errorf("timeout aguardando Highcharts carregar: %v", err)
	}

	// Extrair categorias (datas) e dados (valores)
	// Script para retornar um objeto com ambos
	script := `
		var chart = window.Highcharts.charts[0];
		var categories = chart.xAxis[0].categories;
		var data = chart.series[0].options.data;
		return {categories: categories, data: data};
	`

	res, err := wd.ExecuteScript(script, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar JS para extrair dados: %v", err)
	}

	resultMap, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("formato de retorno do JS inválido")
	}

	catsInterface, ok1 := resultMap["categories"].([]interface{})
	dataInterface, ok2 := resultMap["data"].([]interface{})

	if !ok1 || !ok2 {
		// Tentar fallback se data for objetos {name: "Jan 2023", y: 0.5}
		// Mas assumindo que categories existe baseado na investigação
		return nil, fmt.Errorf("não foi possível converter categories ou data para array")
	}

	if len(catsInterface) != len(dataInterface) {
		return nil, fmt.Errorf("tamanho de categorias (%d) e dados (%d) não batem", len(catsInterface), len(dataInterface))
	}

	ipcaMap := make(map[string]float64)

	for i, catVal := range catsInterface {
		dateStr, ok := catVal.(string) // Ex: "janeiro 1980"
		if !ok {
			continue
		}

		valVal := dataInterface[i]
		var valFloat float64

		// Selenium pode retornar int ou float dependendo do valor
		switch v := valVal.(type) {
		case float64:
			valFloat = v
		case int64:
			valFloat = float64(v)
		case int:
			valFloat = float64(v)
		default:
			// Tentar string parse se necessário, mas Highcharts data costuma ser numérico
			continue
		}

		// Parse da data "janeiro 1980" -> "1980-01"
		parts := strings.Split(dateStr, " ")
		if len(parts) != 2 {
			continue
		}
		mesNome := strings.ToLower(parts[0])
		ano := parts[1]

		mesNum, ok := monthMap[mesNome]
		if !ok {
			continue
		}

		key := fmt.Sprintf("%s-%s", ano, mesNum)
		ipcaMap[key] = valFloat
	}

	log.Printf("[Scraping IPCA] Extraídos %d registros de IPCA.", len(ipcaMap))
	return ipcaMap, nil
}

func tryToClick(wd selenium.WebDriver, by, selector string) {
	el, err := wd.FindElement(by, selector)
	if err == nil {
		el.Click()
		log.Printf("Popup/Banner (%s) clicado com sucesso.", selector)
		time.Sleep(500 * time.Millisecond) // Espera a ação
	} else {
		log.Printf("Popup/Banner (%s) não encontrado. Continuando...", selector)
	}
}

func scrapeCopomAtas(existingMeetings map[int]bool, onSave func(CopomAta) error) error {
	service, err := selenium.NewChromeDriverService("./chromedriver-linux64/chromedriver", seleniumPort)
	if err != nil {
		log.Printf("Erro ao iniciar o ChromeDriverService. Verifique o caminho do chromedriver.")
		return err
	}
	defer service.Stop()

	caps := selenium.Capabilities{
		"browserName":      "chrome",
		"pageLoadStrategy": "eager",
	}
	chromeCaps := chrome.Capabilities{
		Args: []string{
			"--headless",
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-gpu",
			"--window-size=1920,1080",
			"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
	}
	caps.AddChrome(chromeCaps)
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", seleniumPort))
	if err != nil {
		return err
	}
	defer wd.Quit()

	// 1. Obter dados do IPCA (histórico completo)
	ipcaMap, err := getIPCAPorScraping(wd)
	if err != nil {
		log.Printf("AVISO: Falha ao obter dados do IPCA: %v. O campo valor_ipca ficará vazio.", err)
		ipcaMap = make(map[string]float64)
	}

	// 2. Navegar para a lista de atas
	if err := wd.Get(copomListURL); err != nil {
		return err
	}

	log.Println("Aguardando o conteúdo dinâmico carregar (lista de atas)...")
	waitTimeout := 15 * time.Second
	firstLinkSelector := "//div[contains(@class, 'resultados-relacionados')]//h4/a"

	err = wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		el, err := wd.FindElement(selenium.ByXPATH, firstLinkSelector)
		if err != nil {
			return false, nil
		}
		return el.IsDisplayed()
	}, waitTimeout)

	if err != nil {
		return fmt.Errorf("timeout: conteúdo (lista de atas) não carregou em %v", waitTimeout)
	}

	linkElements, err := wd.FindElements(selenium.ByXPATH, "//div[contains(@class, 'resultados-relacionados')]//h4/a")
	if err != nil {
		return err
	}

	type LinkInfo struct {
		URL  string
		Text string
	}
	var links []LinkInfo

	for _, el := range linkElements {
		href, _ := el.GetAttribute("href")
		text, _ := el.Text()
		if href != "" {
			links = append(links, LinkInfo{URL: href, Text: text})
		}
	}

	if len(links) == 0 {
		return fmt.Errorf("nenhum link de ata foi encontrado")
	}

	log.Printf("Encontrados %d links. Iniciando processamento...", len(links))

	for _, link := range links {
		// Tentar extrair número da reunião do texto do link para pular se já existir
		num := extractMeetingNumber(link.Text)
		if num != 0 && existingMeetings[num] {
			log.Printf("Ata %d já existe. Pulando...", num)
			continue
		}

		log.Printf("-----------------------------------------------------")
		log.Printf("Processando Ata URL: %s (Texto: %s)", link.URL, link.Text)

		if err := wd.Get(link.URL); err != nil {
			log.Printf("AVISO: Falha ao abrir a URL %s. Pulando...", link.URL)
			continue
		}

		err = wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
			_, err := wd.FindElement(selenium.ByID, "atacompleta")
			return err == nil, nil
		}, waitTimeout)
		if err != nil {
			log.Printf("AVISO: Conteúdo da ata não carregou em %s. Pulando...", link.URL)
			continue
		}

		titleElement, err := wd.FindElement(selenium.ByTagName, "h3")
		if err != nil {
			log.Printf("AVISO: Não foi possível encontrar o título (h3) na URL %s. Pulando...", link.URL)
			continue
		}
		titulo, _ := titleElement.Text()

		contentElement, _ := wd.FindElement(selenium.ByID, "atacompleta")
		conteudo, _ := contentElement.Text()

		ata := CopomAta{
			URL:           link.URL,
			Titulo:        strings.TrimSpace(titulo),
			Conteudo:      conteudo,
			NumeroReuniao: extractMeetingNumber(titulo),
		}

		// Se não conseguimos extrair do link, usamos o do título.
		// Se ainda assim já existir, não deveríamos salvar?
		// Mas já gastamos o tempo de scraping. Vamos salvar para garantir ou pular?
		// Se já existe, melhor pular antes. Mas se chegamos aqui, é porque não detectamos antes ou não existia.
		if ata.NumeroReuniao != 0 && existingMeetings[ata.NumeroReuniao] {
			log.Printf("Ata %d detectada após scraping (título), mas já existe na base. Não salvando duplicata.", ata.NumeroReuniao)
			continue
		}

		var dataReuniao string

		dataReuniao, err = extractDateFromURL(link.URL)
		if err != nil {
			log.Printf("AVISO: Não foi possível extrair data da URL (%s). Tentando extrair do conteúdo...", link.URL)
			dataReuniao, err = extractDateFromContent(conteudo)
		}

		if err != nil {
			log.Printf("AVISO: FALHA AO EXTRAIR DATA: Não foi possível extrair nem da URL nem do conteúdo: %s. %v", link.URL, err)
		} else {
			ata.DataReuniao = dataReuniao
			log.Printf("Data da reunião extraída: %s", dataReuniao)

			dolar, err := getDolarPorScraping(wd, dataReuniao)
			if err != nil {
				log.Printf("AVISO: Falha ao fazer scraping do dólar para data %s: %v", dataReuniao, err)
			} else {
				ata.ValorDolar = dolar
				log.Printf("Dólar (Scraping) encontrado: %.4f", dolar)
			}

			// Buscar IPCA correspondente (YYYY-MM)
			if len(dataReuniao) >= 7 {
				mesAno := dataReuniao[:7] // "2023-10"
				if val, ok := ipcaMap[mesAno]; ok {
					ata.ValorIPCA = val
					log.Printf("IPCA encontrado para %s: %.2f%%", mesAno, val)
				} else {
					log.Printf("AVISO: IPCA não encontrado para o mês %s", mesAno)
				}
			}
		}

		// Salvar imediatamente
		if err := onSave(ata); err != nil {
			log.Printf("ERRO CRÍTICO: Falha ao salvar ata %d: %v", ata.NumeroReuniao, err)
		} else {
			log.Printf("Ata %d salva com sucesso.", ata.NumeroReuniao)
			// Atualizar mapa em memória para evitar reprocessamento futuro na mesma execução (se houver duplicatas nos links)
			if ata.NumeroReuniao != 0 {
				existingMeetings[ata.NumeroReuniao] = true
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	log.Printf("-----------------------------------------------------")
	return nil
}
