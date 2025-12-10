.PHONY: run build clean deps download-driver

BINARY_NAME=copom-crawler

run:
	rm -f debug_calendar_fail_*.png
	go run . -mode=serve

run-scrape:
	go run . -mode=scrape

# Defina sua chave aqui ou passe via linha de comando: make run-enrich GEMINI_API_KEY=sua_chave
GEMINI_API_KEY ?= ""

run-enrich:
	GEMINI_API_KEY=$(GEMINI_API_KEY) go run . -mode=enrich

build:
	go build -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)
	rm -f *.png *.html dataset_raw.json dataset_enriched.json

deps:
	go mod download

download-driver:
	@echo "Fetching latest ChromeDriver URL..."
	@URL=$$(curl -s "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json" | grep -o 'https://[^"]*chromedriver-linux64.zip' | head -n 1); \
	if [ -z "$$URL" ]; then echo "Error: Could not find URL"; exit 1; fi; \
	echo "Downloading $$URL..."; \
	curl -o chromedriver.zip "$$URL"; \
	unzip -o chromedriver.zip; \
	rm chromedriver.zip; \
	chmod +x chromedriver-linux64/chromedriver; \
	echo "ChromeDriver updated."
