#!/usr/bin/env python3
"""
Graficos de analise do dataset_raw.json - Atas do COPOM
"""

import json
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
from datetime import datetime
from collections import Counter
import numpy as np

plt.style.use('seaborn-v0_8-whitegrid')
plt.rcParams['figure.figsize'] = (12, 6)
plt.rcParams['font.size'] = 10
plt.rcParams['axes.titlesize'] = 12
plt.rcParams['axes.labelsize'] = 10

def carregar_dados(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        return json.load(f)

def preparar_dados(dados):
    """Prepara dados com datas parseadas e ordenados cronologicamente"""
    registros = []
    for item in dados:
        data_str = item.get('data_reuniao', '')
        if data_str and len(data_str) == 10:
            try:
                data = datetime.strptime(data_str, '%Y-%m-%d')
                registros.append({
                    'data': data,
                    'numero': item.get('numero_reuniao', 0),
                    'dolar': item.get('valor_dolar', 0),
                    'ipca': item.get('valor_ipca'),
                    'tamanho': len(item.get('conteudo', '')),
                    'falha': item.get('falha_no_parse', False)
                })
            except:
                pass
    return sorted(registros, key=lambda x: x['data'])

def grafico_evolucao_dolar(registros):
    """1. Evolucao historica do Dolar PTAX"""
    fig, ax = plt.subplots(figsize=(14, 6))

    datas = [r['data'] for r in registros if r['dolar'] > 0]
    dolares = [r['dolar'] for r in registros if r['dolar'] > 0]

    ax.plot(datas, dolares, 'b-', linewidth=1.5, alpha=0.8)
    ax.fill_between(datas, dolares, alpha=0.3)

    ax.set_title('Evolucao do Dolar PTAX nas Datas das Reunioes do COPOM (1998-2025)')
    ax.set_xlabel('Data')
    ax.set_ylabel('Valor do Dolar (R$)')

    ax.xaxis.set_major_locator(mdates.YearLocator(2))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y'))
    plt.xticks(rotation=45)

    # Anotacoes de eventos importantes
    ax.axhline(y=max(dolares), color='r', linestyle='--', alpha=0.5, label=f'Maximo: R${max(dolares):.2f}')
    ax.axhline(y=min(dolares), color='g', linestyle='--', alpha=0.5, label=f'Minimo: R${min(dolares):.2f}')
    ax.legend(loc='upper left')

    plt.tight_layout()
    plt.savefig('grafico_1_evolucao_dolar.png', dpi=150)
    plt.close()
    print("Salvo: grafico_1_evolucao_dolar.png")

def grafico_evolucao_ipca(registros):
    """2. Evolucao historica do IPCA mensal"""
    fig, ax = plt.subplots(figsize=(14, 6))

    datas = [r['data'] for r in registros if r['ipca'] is not None]
    ipcas = [r['ipca'] for r in registros if r['ipca'] is not None]

    colors = ['red' if v > 1.0 else 'orange' if v > 0.5 else 'green' if v >= 0 else 'blue' for v in ipcas]

    ax.bar(datas, ipcas, width=20, color=colors, alpha=0.7, edgecolor='none')

    ax.axhline(y=0, color='black', linestyle='-', linewidth=0.5)
    ax.axhline(y=0.5, color='orange', linestyle='--', alpha=0.5, label='Meta aproximada (0.5%/mes)')

    ax.set_title('IPCA Mensal nas Datas das Reunioes do COPOM (1998-2025)')
    ax.set_xlabel('Data')
    ax.set_ylabel('IPCA (%)')

    ax.xaxis.set_major_locator(mdates.YearLocator(2))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y'))
    plt.xticks(rotation=45)
    ax.legend(loc='upper right')

    plt.tight_layout()
    plt.savefig('grafico_2_evolucao_ipca.png', dpi=150)
    plt.close()
    print("Salvo: grafico_2_evolucao_ipca.png")

def grafico_atas_por_ano(registros):
    """3. Quantidade de atas por ano"""
    fig, ax = plt.subplots(figsize=(14, 6))

    anos = Counter(r['data'].year for r in registros)
    anos_sorted = sorted(anos.items())

    anos_list = [str(a[0]) for a in anos_sorted]
    counts = [a[1] for a in anos_sorted]

    colors = plt.cm.Blues(np.linspace(0.4, 0.9, len(anos_list)))
    bars = ax.bar(anos_list, counts, color=colors, edgecolor='darkblue', linewidth=0.5)

    # Adicionar valores nas barras
    for bar, count in zip(bars, counts):
        ax.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.2,
                str(count), ha='center', va='bottom', fontsize=9)

    ax.set_title('Quantidade de Reunioes do COPOM por Ano')
    ax.set_xlabel('Ano')
    ax.set_ylabel('Numero de Atas')
    plt.xticks(rotation=45)

    # Destacar gap 2016-2020
    ax.axvspan(anos_list.index('2016'), anos_list.index('2020'), alpha=0.2, color='red', label='Gap nos dados')
    ax.legend()

    plt.tight_layout()
    plt.savefig('grafico_3_atas_por_ano.png', dpi=150)
    plt.close()
    print("Salvo: grafico_3_atas_por_ano.png")

def grafico_correlacao_dolar_ipca(registros):
    """4. Scatter plot: Correlacao entre Dolar e IPCA"""
    fig, ax = plt.subplots(figsize=(10, 8))

    dados_validos = [(r['dolar'], r['ipca'], r['data'].year) for r in registros
                     if r['dolar'] > 0 and r['ipca'] is not None]

    dolares = [d[0] for d in dados_validos]
    ipcas = [d[1] for d in dados_validos]
    anos = [d[2] for d in dados_validos]

    # Colorir por decada
    colors = []
    for ano in anos:
        if ano < 2005:
            colors.append('red')
        elif ano < 2010:
            colors.append('orange')
        elif ano < 2015:
            colors.append('green')
        elif ano < 2020:
            colors.append('blue')
        else:
            colors.append('purple')

    scatter = ax.scatter(dolares, ipcas, c=colors, alpha=0.6, s=50, edgecolors='white', linewidth=0.5)

    # Linha de tendencia
    z = np.polyfit(dolares, ipcas, 1)
    p = np.poly1d(z)
    x_line = np.linspace(min(dolares), max(dolares), 100)
    ax.plot(x_line, p(x_line), 'k--', alpha=0.5, label=f'Tendencia linear')

    ax.set_title('Correlacao entre Dolar e IPCA\n(Cor indica periodo)')
    ax.set_xlabel('Valor do Dolar (R$)')
    ax.set_ylabel('IPCA (%)')

    # Legenda manual
    from matplotlib.patches import Patch
    legend_elements = [
        Patch(facecolor='red', label='1998-2004'),
        Patch(facecolor='orange', label='2005-2009'),
        Patch(facecolor='green', label='2010-2014'),
        Patch(facecolor='blue', label='2015-2019'),
        Patch(facecolor='purple', label='2020-2025'),
    ]
    ax.legend(handles=legend_elements, loc='upper right')

    # Calcular e mostrar correlacao
    n = len(dados_validos)
    mean_d = sum(dolares) / n
    mean_i = sum(ipcas) / n
    num = sum((d - mean_d) * (i - mean_i) for d, i in zip(dolares, ipcas))
    den_d = sum((d - mean_d) ** 2 for d in dolares) ** 0.5
    den_i = sum((i - mean_i) ** 2 for i in ipcas) ** 0.5
    corr = num / (den_d * den_i) if den_d > 0 and den_i > 0 else 0
    ax.text(0.05, 0.95, f'Correlacao: {corr:.3f}', transform=ax.transAxes, fontsize=11,
            verticalalignment='top', bbox=dict(boxstyle='round', facecolor='wheat', alpha=0.5))

    plt.tight_layout()
    plt.savefig('grafico_4_correlacao_dolar_ipca.png', dpi=150)
    plt.close()
    print("Salvo: grafico_4_correlacao_dolar_ipca.png")

def grafico_tamanho_atas(registros):
    """5. Evolucao do tamanho das atas ao longo do tempo"""
    fig, ax = plt.subplots(figsize=(14, 6))

    datas = [r['data'] for r in registros]
    tamanhos = [r['tamanho'] / 1000 for r in registros]  # Em milhares

    ax.scatter(datas, tamanhos, c='steelblue', alpha=0.6, s=30)

    # Media movel (janela de 10)
    window = 10
    if len(tamanhos) > window:
        media_movel = []
        datas_mm = []
        for i in range(window, len(tamanhos)):
            media_movel.append(sum(tamanhos[i-window:i]) / window)
            datas_mm.append(datas[i])
        ax.plot(datas_mm, media_movel, 'r-', linewidth=2, label=f'Media movel ({window} atas)')

    ax.set_title('Evolucao do Tamanho das Atas do COPOM ao Longo do Tempo')
    ax.set_xlabel('Data')
    ax.set_ylabel('Tamanho (milhares de caracteres)')

    ax.xaxis.set_major_locator(mdates.YearLocator(2))
    ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y'))
    plt.xticks(rotation=45)
    ax.legend()

    plt.tight_layout()
    plt.savefig('grafico_5_tamanho_atas.png', dpi=150)
    plt.close()
    print("Salvo: grafico_5_tamanho_atas.png")

def grafico_boxplot_ipca_decada(registros):
    """6. Boxplot do IPCA por decada"""
    fig, ax = plt.subplots(figsize=(10, 6))

    decadas = {}
    for r in registros:
        if r['ipca'] is not None:
            decada = (r['data'].year // 10) * 10
            if decada not in decadas:
                decadas[decada] = []
            decadas[decada].append(r['ipca'])

    labels = sorted(decadas.keys())
    data = [decadas[d] for d in labels]
    labels_str = [f"{d}s" for d in labels]

    bp = ax.boxplot(data, labels=labels_str, patch_artist=True)

    colors = plt.cm.Set3(np.linspace(0, 1, len(labels)))
    for patch, color in zip(bp['boxes'], colors):
        patch.set_facecolor(color)

    ax.set_title('Distribuicao do IPCA por Decada')
    ax.set_xlabel('Decada')
    ax.set_ylabel('IPCA (%)')
    ax.axhline(y=0, color='black', linestyle='-', linewidth=0.5)

    plt.tight_layout()
    plt.savefig('grafico_6_boxplot_ipca_decada.png', dpi=150)
    plt.close()
    print("Salvo: grafico_6_boxplot_ipca_decada.png")

def grafico_dolar_ipca_dual(registros):
    """7. Grafico dual: Dolar e IPCA no mesmo grafico"""
    fig, ax1 = plt.subplots(figsize=(14, 6))

    datas = [r['data'] for r in registros if r['dolar'] > 0 and r['ipca'] is not None]
    dolares = [r['dolar'] for r in registros if r['dolar'] > 0 and r['ipca'] is not None]
    ipcas = [r['ipca'] for r in registros if r['dolar'] > 0 and r['ipca'] is not None]

    # Eixo 1: Dolar
    color1 = 'tab:blue'
    ax1.set_xlabel('Data')
    ax1.set_ylabel('Dolar (R$)', color=color1)
    line1 = ax1.plot(datas, dolares, color=color1, linewidth=1.5, label='Dolar PTAX')
    ax1.tick_params(axis='y', labelcolor=color1)
    ax1.fill_between(datas, dolares, alpha=0.2, color=color1)

    # Eixo 2: IPCA
    ax2 = ax1.twinx()
    color2 = 'tab:red'
    ax2.set_ylabel('IPCA (%)', color=color2)
    line2 = ax2.plot(datas, ipcas, color=color2, linewidth=1.5, alpha=0.8, label='IPCA')
    ax2.tick_params(axis='y', labelcolor=color2)
    ax2.axhline(y=0, color='gray', linestyle='--', linewidth=0.5)

    ax1.set_title('Evolucao Comparativa: Dolar PTAX vs IPCA (1998-2025)')

    ax1.xaxis.set_major_locator(mdates.YearLocator(2))
    ax1.xaxis.set_major_formatter(mdates.DateFormatter('%Y'))
    plt.xticks(rotation=45)

    # Legenda combinada
    lines = line1 + line2
    labels = [l.get_label() for l in lines]
    ax1.legend(lines, labels, loc='upper left')

    plt.tight_layout()
    plt.savefig('grafico_7_dolar_ipca_dual.png', dpi=150)
    plt.close()
    print("Salvo: grafico_7_dolar_ipca_dual.png")

def grafico_completude(dados):
    """8. Grafico de completude dos dados"""
    fig, ax = plt.subplots(figsize=(10, 6))

    campos = ['numero_reuniao', 'url', 'titulo', 'data_reuniao', 'valor_dolar', 'valor_ipca', 'conteudo']
    total = len(dados)

    completos = []
    for campo in campos:
        count = sum(1 for item in dados if item.get(campo) not in [None, '', 0])
        completos.append((count / total) * 100)

    colors = ['green' if c >= 95 else 'orange' if c >= 80 else 'red' for c in completos]

    bars = ax.barh(campos, completos, color=colors, edgecolor='darkgray')

    ax.set_xlim(0, 105)
    ax.axvline(x=95, color='green', linestyle='--', alpha=0.5, label='95% (excelente)')
    ax.axvline(x=80, color='orange', linestyle='--', alpha=0.5, label='80% (bom)')

    for bar, pct in zip(bars, completos):
        ax.text(bar.get_width() + 1, bar.get_y() + bar.get_height()/2,
                f'{pct:.1f}%', va='center', fontsize=10)

    ax.set_title('Completude dos Dados por Campo')
    ax.set_xlabel('Porcentagem de Registros Preenchidos (%)')
    ax.legend(loc='lower right')

    plt.tight_layout()
    plt.savefig('grafico_8_completude.png', dpi=150)
    plt.close()
    print("Salvo: grafico_8_completude.png")

def main():
    print("Gerando graficos do dataset COPOM...")
    print("=" * 50)

    dados = carregar_dados('dataset_raw.json')
    registros = preparar_dados(dados)

    print(f"Registros carregados: {len(dados)}")
    print(f"Registros com data valida: {len(registros)}")
    print()

    grafico_evolucao_dolar(registros)
    grafico_evolucao_ipca(registros)
    grafico_atas_por_ano(registros)
    grafico_correlacao_dolar_ipca(registros)
    grafico_tamanho_atas(registros)
    grafico_boxplot_ipca_decada(registros)
    grafico_dolar_ipca_dual(registros)
    grafico_completude(dados)

    print()
    print("=" * 50)
    print("Todos os graficos foram gerados com sucesso!")
    print("Arquivos PNG salvos no diretorio atual.")

if __name__ == '__main__':
    main()
