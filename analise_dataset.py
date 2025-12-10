#!/usr/bin/env python3
"""
Analise completa do dataset_raw.json - Atas do COPOM
"""

import json
import statistics
from collections import Counter, defaultdict
from datetime import datetime
import re

def carregar_dados(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        return json.load(f)

def analise_estrutural(dados):
    print("=" * 80)
    print("1. ANALISE ESTRUTURAL DOS DADOS")
    print("=" * 80)

    if not dados:
        print("Dataset vazio!")
        return

    # Atributos disponiveis
    print("\n1.1 ATRIBUTOS (COLUNAS/CAMPOS) DISPONIVEIS:")
    print("-" * 40)
    campos = set()
    for item in dados:
        campos.update(item.keys())
    for campo in sorted(campos):
        print(f"  - {campo}")

    # Tipos de dados
    print("\n1.2 TIPOS DE DADOS:")
    print("-" * 40)
    tipos = {}
    for campo in campos:
        valores_exemplo = []
        for item in dados[:10]:
            if campo in item and item[campo] is not None:
                valores_exemplo.append(type(item[campo]).__name__)
        if valores_exemplo:
            tipo_comum = Counter(valores_exemplo).most_common(1)[0][0]
            tipos[campo] = tipo_comum

    for campo, tipo in sorted(tipos.items()):
        tipo_desc = {
            'str': 'string (texto)',
            'int': 'inteiro (numero)',
            'float': 'float (numero decimal)',
            'bool': 'booleano (verdadeiro/falso)',
            'NoneType': 'nulo',
            'list': 'lista',
            'dict': 'dicionario'
        }.get(tipo, tipo)
        print(f"  - {campo}: {tipo_desc}")

    # Valores nulos ou ausentes
    print("\n1.3 VALORES NULOS OU AUSENTES:")
    print("-" * 40)
    total = len(dados)
    for campo in sorted(campos):
        nulos = sum(1 for item in dados if campo not in item or item.get(campo) is None or item.get(campo) == "")
        if nulos > 0:
            pct = (nulos / total) * 100
            print(f"  - {campo}: {nulos} nulos/vazios ({pct:.1f}%)")

    # Tamanho medio das entradas
    print("\n1.4 TAMANHO MEDIO DAS ENTRADAS (CARACTERES):")
    print("-" * 40)
    for campo in sorted(campos):
        tamanhos = []
        for item in dados:
            if campo in item and item[campo] is not None:
                tamanhos.append(len(str(item[campo])))
        if tamanhos:
            media = statistics.mean(tamanhos)
            minimo = min(tamanhos)
            maximo = max(tamanhos)
            print(f"  - {campo}:")
            print(f"      Media: {media:.1f} chars | Min: {minimo} | Max: {maximo}")

    # Normalizacao e redundancia
    print("\n1.5 NORMALIZACAO E REDUNDANCIA:")
    print("-" * 40)

    # Verificar URLs duplicadas
    urls = [item.get('url', '') for item in dados if item.get('url')]
    urls_duplicadas = [url for url, count in Counter(urls).items() if count > 1]
    print(f"  - URLs duplicadas: {len(urls_duplicadas)}")

    # Verificar numeros de reuniao duplicados
    numeros = [item.get('numero_reuniao', 0) for item in dados]
    numeros_duplicados = [num for num, count in Counter(numeros).items() if count > 1]
    print(f"  - Numeros de reuniao duplicados: {len(numeros_duplicados)}")
    if numeros_duplicados:
        print(f"      Valores: {numeros_duplicados[:10]}...")

    # Verificar consistencia de formato de data
    formatos_data = Counter()
    for item in dados:
        data = item.get('data_reuniao', '')
        if data:
            if re.match(r'\d{4}-\d{2}-\d{2}', data):
                formatos_data['YYYY-MM-DD'] += 1
            elif re.match(r'\d{2}/\d{2}/\d{4}', data):
                formatos_data['DD/MM/YYYY'] += 1
            else:
                formatos_data['outro'] += 1
    print(f"  - Formatos de data encontrados: {dict(formatos_data)}")

    return campos

def analise_estatistica(dados):
    print("\n" + "=" * 80)
    print("2. ANALISE ESTATISTICA DESCRITIVA")
    print("=" * 80)

    # Frequencia de valores unicos para campos categoricos
    print("\n2.1 FREQUENCIA DE VALORES UNICOS:")
    print("-" * 40)

    # Top reunioes por ano
    anos = Counter()
    for item in dados:
        data = item.get('data_reuniao', '')
        if data and len(data) >= 4:
            ano = data[:4]
            anos[ano] += 1
    print("\n  Reunioes por ano:")
    for ano, count in sorted(anos.items()):
        print(f"    {ano}: {count} atas")

    # Estatisticas de numero de reuniao
    print("\n2.2 ESTATISTICAS - NUMERO DA REUNIAO:")
    print("-" * 40)
    numeros = [item.get('numero_reuniao', 0) for item in dados if item.get('numero_reuniao')]
    if numeros:
        print(f"  Media: {statistics.mean(numeros):.1f}")
        print(f"  Mediana: {statistics.median(numeros):.1f}")
        print(f"  Moda: {statistics.mode(numeros)}")
        print(f"  Desvio padrao: {statistics.stdev(numeros):.2f}")
        print(f"  Minimo: {min(numeros)}")
        print(f"  Maximo: {max(numeros)}")
        print(f"  Total de reunioes: {len(numeros)}")

    # Estatisticas de valor do dolar
    print("\n2.3 ESTATISTICAS - VALOR DO DOLAR (PTAX):")
    print("-" * 40)
    dolares = [item.get('valor_dolar', 0) for item in dados if item.get('valor_dolar') and item.get('valor_dolar') > 0]
    if dolares:
        print(f"  Media: R$ {statistics.mean(dolares):.4f}")
        print(f"  Mediana: R$ {statistics.median(dolares):.4f}")
        print(f"  Desvio padrao: R$ {statistics.stdev(dolares):.4f}")
        print(f"  Minimo: R$ {min(dolares):.4f}")
        print(f"  Maximo: R$ {max(dolares):.4f}")
        print(f"  Registros com valor: {len(dolares)}")
    else:
        print("  Nenhum valor de dolar disponivel")

    # Estatisticas de IPCA
    print("\n2.4 ESTATISTICAS - VALOR DO IPCA:")
    print("-" * 40)
    ipcas = [item.get('valor_ipca', 0) for item in dados if item.get('valor_ipca') is not None]
    if ipcas:
        print(f"  Media: {statistics.mean(ipcas):.4f}%")
        print(f"  Mediana: {statistics.median(ipcas):.4f}%")
        if len(ipcas) > 1:
            print(f"  Desvio padrao: {statistics.stdev(ipcas):.4f}%")
        print(f"  Minimo: {min(ipcas):.4f}%")
        print(f"  Maximo: {max(ipcas):.4f}%")
        print(f"  Registros com valor: {len(ipcas)}")
    else:
        print("  Nenhum valor de IPCA disponivel")

    # Tamanho do conteudo
    print("\n2.5 ESTATISTICAS - TAMANHO DO CONTEUDO (TEXTO DAS ATAS):")
    print("-" * 40)
    tamanhos = [len(item.get('conteudo', '')) for item in dados if item.get('conteudo')]
    if tamanhos:
        print(f"  Media: {statistics.mean(tamanhos):.0f} caracteres")
        print(f"  Mediana: {statistics.median(tamanhos):.0f} caracteres")
        print(f"  Desvio padrao: {statistics.stdev(tamanhos):.0f} caracteres")
        print(f"  Minimo: {min(tamanhos)} caracteres")
        print(f"  Maximo: {max(tamanhos)} caracteres")
        print(f"  Atas com conteudo: {len(tamanhos)}")

    # Distribuicao de falha_no_parse
    print("\n2.6 DISTRIBUICAO - FALHA NO PARSE:")
    print("-" * 40)
    falhas = Counter(item.get('falha_no_parse', False) for item in dados)
    for status, count in falhas.items():
        pct = (count / len(dados)) * 100
        print(f"  {status}: {count} ({pct:.1f}%)")

    # Correlacao simples entre dolar e IPCA
    print("\n2.7 CORRELACAO ENTRE ATRIBUTOS:")
    print("-" * 40)
    pares = [(item.get('valor_dolar', 0), item.get('valor_ipca', 0))
             for item in dados
             if item.get('valor_dolar') and item.get('valor_ipca') is not None]

    if len(pares) > 2:
        dolares_p = [p[0] for p in pares]
        ipcas_p = [p[1] for p in pares]

        # Calculo manual de correlacao de Pearson
        n = len(pares)
        mean_d = sum(dolares_p) / n
        mean_i = sum(ipcas_p) / n

        numerador = sum((d - mean_d) * (i - mean_i) for d, i in pares)
        denom_d = sum((d - mean_d) ** 2 for d in dolares_p) ** 0.5
        denom_i = sum((i - mean_i) ** 2 for i in ipcas_p) ** 0.5

        if denom_d > 0 and denom_i > 0:
            correlacao = numerador / (denom_d * denom_i)
            print(f"  Correlacao Dolar x IPCA: {correlacao:.4f}")
            if abs(correlacao) < 0.3:
                interpretacao = "fraca"
            elif abs(correlacao) < 0.7:
                interpretacao = "moderada"
            else:
                interpretacao = "forte"
            direcao = "positiva" if correlacao > 0 else "negativa"
            print(f"  Interpretacao: correlacao {interpretacao} {direcao}")
    else:
        print("  Dados insuficientes para calcular correlacao")

def analise_qualidade(dados):
    print("\n" + "=" * 80)
    print("3. ANALISE DE QUALIDADE DOS DADOS")
    print("=" * 80)

    # Valores inconsistentes
    print("\n3.1 VALORES INCONSISTENTES:")
    print("-" * 40)

    # Verificar titulos muito diferentes
    titulos = [item.get('titulo', '') for item in dados if item.get('titulo')]
    titulos_normalizados = [t.lower().strip() for t in titulos]
    variacoes_titulo = Counter(titulos_normalizados)
    print(f"  Variacoes de titulo encontradas: {len(variacoes_titulo)}")
    if len(variacoes_titulo) > 1:
        print("  Top 5 variacoes:")
        for titulo, count in variacoes_titulo.most_common(5):
            print(f"    '{titulo[:60]}...' : {count} ocorrencias")

    # Verificar URLs com padroes diferentes
    urls = [item.get('url', '') for item in dados if item.get('url')]
    dominios = Counter()
    for url in urls:
        match = re.search(r'https?://([^/]+)', url)
        if match:
            dominios[match.group(1)] += 1
    print(f"\n  Dominios nas URLs: {dict(dominios)}")

    # Outliers
    print("\n3.2 VALORES FORA DE PADRAO (OUTLIERS):")
    print("-" * 40)

    # Outliers no dolar
    dolares = [item.get('valor_dolar', 0) for item in dados if item.get('valor_dolar') and item.get('valor_dolar') > 0]
    if dolares:
        q1 = sorted(dolares)[len(dolares) // 4]
        q3 = sorted(dolares)[3 * len(dolares) // 4]
        iqr = q3 - q1
        limite_inferior = q1 - 1.5 * iqr
        limite_superior = q3 + 1.5 * iqr
        outliers_dolar = [d for d in dolares if d < limite_inferior or d > limite_superior]
        print(f"  Outliers no valor do dolar: {len(outliers_dolar)}")
        if outliers_dolar:
            print(f"    Limites: [{limite_inferior:.2f}, {limite_superior:.2f}]")
            print(f"    Valores outliers: {sorted(outliers_dolar)[:5]}...")

    # Outliers no IPCA
    ipcas = [item.get('valor_ipca', 0) for item in dados if item.get('valor_ipca') is not None]
    if ipcas:
        q1 = sorted(ipcas)[len(ipcas) // 4]
        q3 = sorted(ipcas)[3 * len(ipcas) // 4]
        iqr = q3 - q1
        limite_inferior = q1 - 1.5 * iqr
        limite_superior = q3 + 1.5 * iqr
        outliers_ipca = [i for i in ipcas if i < limite_inferior or i > limite_superior]
        print(f"  Outliers no IPCA: {len(outliers_ipca)}")
        if outliers_ipca:
            print(f"    Limites: [{limite_inferior:.4f}, {limite_superior:.4f}]")
            print(f"    Valores outliers: {sorted(outliers_ipca)[:5]}...")

    # Completude dos dados
    print("\n3.3 COMPLETUDE DOS DADOS:")
    print("-" * 40)
    total = len(dados)
    campos = ['numero_reuniao', 'url', 'titulo', 'data_reuniao', 'valor_dolar', 'valor_ipca', 'conteudo', 'falha_no_parse']

    print(f"  {'Campo':<20} {'Preenchidos':<15} {'Vazios':<10} {'% Completo':<12}")
    print("  " + "-" * 57)
    for campo in campos:
        preenchidos = sum(1 for item in dados if item.get(campo) not in [None, '', 0, False] or (campo == 'falha_no_parse' and campo in item))
        vazios = total - preenchidos
        pct = (preenchidos / total) * 100
        print(f"  {campo:<20} {preenchidos:<15} {vazios:<10} {pct:.1f}%")

    # Duplicatas
    print("\n3.4 DUPLICATAS E DADOS REPETIDOS:")
    print("-" * 40)

    # Duplicatas por numero de reuniao
    numeros = [item.get('numero_reuniao', 0) for item in dados]
    duplicatas_numero = {num: count for num, count in Counter(numeros).items() if count > 1}
    print(f"  Numeros de reuniao duplicados: {len(duplicatas_numero)}")
    if duplicatas_numero:
        print(f"    Detalhes: {dict(list(duplicatas_numero.items())[:10])}")

    # Duplicatas por URL
    urls = [item.get('url', '') for item in dados if item.get('url')]
    duplicatas_url = {url: count for url, count in Counter(urls).items() if count > 1}
    print(f"  URLs duplicadas: {len(duplicatas_url)}")

    # Duplicatas de conteudo (hash simples)
    conteudos = [hash(item.get('conteudo', '')) for item in dados if item.get('conteudo')]
    duplicatas_conteudo = sum(1 for _, count in Counter(conteudos).items() if count > 1)
    print(f"  Conteudos possivelmente duplicados: {duplicatas_conteudo}")

    # Registros totalmente duplicados
    registros_str = [json.dumps(item, sort_keys=True) for item in dados]
    registros_duplicados = sum(1 for _, count in Counter(registros_str).items() if count > 1)
    print(f"  Registros totalmente duplicados: {registros_duplicados}")

    # Analise temporal
    print("\n3.5 ANALISE TEMPORAL (GAPS NAS DATAS):")
    print("-" * 40)
    datas_validas = []
    for item in dados:
        data = item.get('data_reuniao', '')
        if data and re.match(r'\d{4}-\d{2}-\d{2}', data):
            try:
                datas_validas.append(datetime.strptime(data, '%Y-%m-%d'))
            except:
                pass

    if datas_validas:
        datas_validas.sort()
        print(f"  Primeira ata: {datas_validas[0].strftime('%Y-%m-%d')}")
        print(f"  Ultima ata: {datas_validas[-1].strftime('%Y-%m-%d')}")
        print(f"  Periodo coberto: {(datas_validas[-1] - datas_validas[0]).days} dias")

        # Verificar gaps significativos (> 60 dias sem reuniao)
        gaps = []
        for i in range(1, len(datas_validas)):
            diff = (datas_validas[i] - datas_validas[i-1]).days
            if diff > 60:
                gaps.append((datas_validas[i-1].strftime('%Y-%m-%d'),
                            datas_validas[i].strftime('%Y-%m-%d'), diff))

        print(f"  Gaps significativos (>60 dias): {len(gaps)}")
        if gaps:
            print("  Maiores gaps:")
            for inicio, fim, dias in sorted(gaps, key=lambda x: -x[2])[:5]:
                print(f"    {inicio} -> {fim}: {dias} dias")

def resumo_final(dados):
    print("\n" + "=" * 80)
    print("RESUMO EXECUTIVO")
    print("=" * 80)

    total = len(dados)
    com_conteudo = sum(1 for item in dados if item.get('conteudo'))
    com_dolar = sum(1 for item in dados if item.get('valor_dolar') and item.get('valor_dolar') > 0)
    com_ipca = sum(1 for item in dados if item.get('valor_ipca') is not None)
    com_falha = sum(1 for item in dados if item.get('falha_no_parse'))

    print(f"""
  Total de atas no dataset: {total}
  Atas com conteudo: {com_conteudo} ({com_conteudo/total*100:.1f}%)
  Atas com valor do dolar: {com_dolar} ({com_dolar/total*100:.1f}%)
  Atas com IPCA: {com_ipca} ({com_ipca/total*100:.1f}%)
  Atas com falha no parse: {com_falha} ({com_falha/total*100:.1f}%)

  QUALIDADE GERAL: {'BOA' if com_conteudo/total > 0.9 else 'MODERADA' if com_conteudo/total > 0.7 else 'PRECISA ATENCAO'}
""")

def main():
    print("ANALISE COMPLETA DO DATASET - ATAS DO COPOM")
    print("Arquivo: dataset_raw.json")
    print("Data da analise:", datetime.now().strftime('%Y-%m-%d %H:%M:%S'))
    print()

    dados = carregar_dados('dataset_raw.json')
    print(f"Carregados {len(dados)} registros.\n")

    analise_estrutural(dados)
    analise_estatistica(dados)
    analise_qualidade(dados)
    resumo_final(dados)

if __name__ == '__main__':
    main()
