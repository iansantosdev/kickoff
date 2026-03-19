# kickoff

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/iansantosdev/kickoff)](https://goreportcard.com/report/github.com/iansantosdev/kickoff)
[![License: MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

[English](README.md) | **Português (pt-BR)**

`kickoff` é uma CLI em Go pensada para quem acompanha futebol e passa boa parte do tempo no terminal. Ela oferece uma forma fluida de acompanhar placares ao vivo, próximos jogos e transmissões de TV sem interromper o fluxo de trabalho.

## Destaques

- Busca por time com seleção interativa quando há múltiplos resultados.
- Exibição de próximos jogos e resultados recentes.
- Consulta de vários times em uma única execução.
- Navegação por competição ou liga.
- Listagem de partidas em destaque para períodos relativos como hoje, amanhã e semana.
- Resolução de canais de TV para um país específico, com fallback automático baseado no sistema.
- Uso da CLI em inglês ou português (`pt-BR`), com flags longas específicas para cada idioma.

## Requisitos

- Go `1.26.1` ou superior
- Acesso à internet para buscar os dados das partidas

## Instalação

### Instalar com `go install`

```bash
go install github.com/iansantosdev/kickoff/cmd/kickoff@latest
```

### Build local

```bash
git clone https://github.com/iansantosdev/kickoff.git
cd kickoff
go build -o bin/kickoff ./cmd/kickoff
```

Se você usa [`just`](https://github.com/casey/just), também pode gerar um build de release com:

```bash
just build-release
```

## Uso rápido

```bash
# Comportamento padrão: mostra o próximo jogo do Fluminense
kickoff

# Busca um time e mostra seu próximo jogo
kickoff --time "Real Madrid"

# Mostra os próximos 3 jogos
kickoff --time "Arsenal" --proximos 3

# Mostra os últimos 5 jogos
kickoff --time "Barcelona" --ultimos 5

# Consulta vários times na mesma execução
kickoff --time "Flamengo, Palmeiras, Liverpool"

# Mostra jogos de uma competição ao longo da próxima semana
kickoff --liga "UEFA Champions League"

# Mostra os destaques de hoje
kickoff --destaques hoje

# Filtra destaques por liga
kickoff --destaques hoje --liga "Premier League"

# Filtra destaques por time
kickoff --destaques semana --time "Bayern"

# Resolve transmissões de TV para um país específico
kickoff --time "Inter Miami" --pais US
```

Para ver a ajuda completa:

```bash
kickoff -h
```

## Referência da CLI

| Flag | Aliases | Descrição | Padrão |
| --- | --- | --- | --- |
| `--time` | `-t` | Nome do time a ser buscado | `Fluminense` |
| `--proximos` | `-n` | Quantidade de próximos jogos a mostrar | `1` no modo time quando `--ultimos` não é usado |
| `--ultimos` | `-l` | Quantidade de jogos passados a mostrar | `0` |
| `--liga` | `-L` | Filtra por nome da competição ou liga | vazio |
| `--destaques` | `-f` | Mostra jogos em destaque para um período relativo | vazio |
| `--pais` | `-c` | Código do país usado para transmissões de TV | `KICKOFF_COUNTRY` ou detecção automática |
| `--idioma` | `-g` | Idioma da interface (`en`, `pt-BR`) | `KICKOFF_LANG` ou idioma do sistema |
| `--detalhado` | `-v` | Exibe logs detalhados | `false` |

### Valores aceitos em `--destaques`

Períodos suportados:

- `hoje`, `amanhã`, `semana`, `ontem`

`--destaques` não pode ser combinado com `--proximos` ou `--ultimos`.

## Variáveis de ambiente

Você pode persistir preferências com:

```bash
export KICKOFF_LANG=pt-BR
export KICKOFF_COUNTRY=BR
```

Quando `KICKOFF_COUNTRY` não está definido, o `kickoff` tenta inferir o país a partir do idioma configurado e, depois, da variável `LANG` do sistema. A normalização de países aceita códigos ISO alpha-2, abreviações esportivas comuns e nomes de países.

## Fluxos suportados

Hoje o `kickoff` cobre quatro padrões principais de uso:

1. Modo time: busca partidas de um ou mais times.
2. Modo liga: lista partidas de uma competição, com desambiguação interativa quando necessário.
3. Modo destaques: mostra jogos de competições principais em um período relativo.
4. Modo combinado: filtra destaques por liga e/ou por time.

## Desenvolvimento

### Estrutura do projeto

```text
cmd/kickoff         # ponto de entrada da CLI
internal/cli        # fluxos de execução, interação e formatação de saída
internal/domain     # modelos de domínio
internal/i18n       # traduções e normalização de países
internal/sofascore  # cliente HTTP e mapeamento da API
```

### Comandos úteis

Se você usa `just`, estes comandos estão disponíveis:

| Comando | Descrição |
| --- | --- |
| `just run -- <args>` | Executa a CLI em modo de desenvolvimento |
| `just build` | Gera `bin/kickoff` |
| `just build-release` | Executa checks e gera um build otimizado |
| `just lint` | Executa `golangci-lint` |
| `just test` | Executa a suíte de testes |
| `just test-race` | Executa testes com detector de race |
| `just vet` | Executa `go vet` |
| `just fmt-check` | Verifica formatação dos arquivos Go |
| `just check` | Executa lint e testes |
| `just qa` | Executa checagens de formatação, vet, lint e testes |
| `just build-obfuscated` | Gera um build ofuscado com `garble` |

Sem `just`, você pode rodar:

```bash
go test ./...
go vet ./...
golangci-lint run ./...
go run ./cmd/kickoff -h
```

## Aviso

`kickoff` é um projeto open source independente e não possui afiliação com o Sofascore. O acesso aos dados das partidas pode estar sujeito aos termos, limites ou disponibilidade do provedor.

## Licença

Este projeto está licenciado sob a licença MIT. Veja [LICENSE](LICENSE) para mais detalhes.
