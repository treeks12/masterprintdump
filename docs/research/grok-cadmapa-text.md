# Grok Research: CadMapa Text

Fonte: pesquisa delegada ao `grok` em modo somente leitura, sem `--max-turns`.

## Funcoes analisadas

| Funcao | Papel |
|---|---|
| `FUN_004bf12c` | Load do objeto texto no stream ETQ |
| `FUN_004bf340` | Save do objeto texto no stream ETQ |
| `FUN_004bf99c` | Ajusta largura media de caractere para caber no `RECT` |
| `FUN_004c036c` | Orquestra render de texto, prepara fonte e chama saida GDI |
| `FUN_004c0118` | Saida final via metafile, `TextOutA` ou `DrawTextA` |

## Offsets confirmados

| Offset | Campo |
|---|---|
| `+0x0c` | X |
| `+0x10` | Y |
| `+0x18` | Width |
| `+0x1c` | Height |
| `+0x34` | Texto renderizado |
| `+0x38` | Conteudo alternativo/RTF |
| `+0x3c` | Largura media de caractere calculada |
| `+0x40` | Escapement/orientacao |
| `+0x42` | Alinhamento horizontal |
| `+0x44` | Modo texto artistico/metafile |
| `+0x48` | `RECT` cacheado |
| `+0x5c` | `1` usa `DrawTextA`, `0` usa `TextOutA` |

## Render

O caminho de tela copia o `RECT` recebido por `FUN_004c036c` e usa esse retangulo para desenhar. A altura da fonte vem de `RECT.bottom - RECT.top`, nao de heuristica visual.

`FUN_004c0118` escolhe:

| Condicao | Saida |
|---|---|
| preview/impressao normal com `+0x44 == 0` | `FUN_004bfe08` via metafile |
| `+0x5c == 0` | `TextOutA(hdc, rect.left, rect.top, text, len)` |
| `+0x5c != 0` | `DrawTextA(hdc, text, len, &rect, flags)` |

## Fonte e estilo

`CreateFontA` em `FUN_004c036c` usa:

| Parametro | Origem |
|---|---|
| Height | `RECT.bottom - RECT.top` |
| Width | campo `+0x3c`, calculado por `FUN_004bf99c` |
| Escapement | `+0x40` |
| Face | descritor de fonte do contexto |
| Bold | bit `0x01` do style byte |
| Italic | bit `0x02` |
| Underline | bit `0x04` |
| Strikeout | bit `0x08` |

## Alinhamento

Campo `+0x42`:

| Valor | Flags | Significado |
|---|---|---|
| `0` | `0x810` | Esquerda |
| `1` | `0x811` | Centro |
| `2` | `0x812` | Direita |

Nao ha evidencia de `DT_VCENTER` no caminho CadMapa analisado.

## Recomendacoes

1. Renomear campos do parser para refletir `Height` e `Width`, pois os nomes atuais `TextW/TextH` estao semanticamente invertidos.
2. Decodificar style byte por bitmask: bold, italic, underline e strikeout.
3. Nao forcar alinhamento `center` em `main.go`; parsear/alimentar alinhamento real quando encontrado.
4. Renderizar texto no canvas com flags equivalentes a `DrawTextA`, sem vertical center por padrao.
5. Em impressao, evoluir de `TextOut` pontual para desenho por caixa/retangulo.
