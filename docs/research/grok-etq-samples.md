# Grok Research: ETQ Samples

Fonte: pesquisa delegada ao `grok` em modo somente leitura, sem `--max-turns`.

Amostras analisadas em `C:\Program Files (x86)\paulimaq\ARQUIVOS`. Unidades: centesimos de mm.

## Padroes confirmados

| Campo | Texto (`tag=1`, `flags=0`) | WMF (`tag=0`, `flags=0x80000008`) |
|---|---|---|
| Marcador | `FE FF FF FF` | `FE FF FF FF` |
| `head0/head1` | `x/y` | `width/height` |
| `post1/post2` | `height/width`, apos terminador `FFFFFFFF` | `x/y`, no bloco 83 bytes antes do `FE` |
| Payload | `u16` length em `+38`, bytes em `+40` | Aldus `D7 CD C6 9A` em `+49` |
| `nextX/nextY` | Coordenadas do proximo objeto | Coordenadas do proximo objeto ou auto-referencia em alguns WMF |
| `style` | Valores observados `4`, `5`, `6` | Geralmente `6` |

## Amostras e observacoes

| Arquivo | Registros uteis | Observacoes |
|---|---:|---|
| `Canelado algodao (Classic Wave Ramado) lunelli.ETQ` | 13 | Boa amostra base para texto, composicao e WMF |
| `ADAR SOFA CANELADO.ETQ` | 10 + falso-positivos | RTF e `FE` dentro de blob WMF |
| `FAVERO.ETQ` | 14 | WMF nao quadrado e texto comum |
| `SLIM.ETQ` | 14 + no vazio | Possui registro com `tln=0` na cadeia |
| `jeans.ETQ` | 11 | Cadeia curta, composicao simples |
| `ALGODAO.ETQ` | 12 + truncado EOF | Registro truncado no final do arquivo |

## Excecoes importantes

1. `ADAR SOFA CANELADO.ETQ` tem falso-positivo `FE` dentro de blob WMF.
2. `SLIM.ETQ` tem no texto vazio com `tln=0`, mas ele participa da cadeia.
3. `ALGODAO.ETQ` tem registro truncado no EOF com texto legivel, sem post-fields completos.
4. Textos numericos como CNPJ podem ter `post1 > post2`; nao usar regra geometrica simplista.
5. WMF pode nao ser quadrado; nao assumir `width == height`.
6. Dedupe por conteudo e perigoso: textos ou WMFs iguais podem ser objetos distintos.

## Testes recomendados

| Teste | Proposito |
|---|---|
| `TestFERecordFields_Lunelli` | Travar campos binarios de um texto conhecido |
| `TestFEWMFPreBlock_Lunelli` | Travar `x/y` de WMF no bloco `offset-83` |
| `TestFEWMFNonSquare_FAVERO` | Garantir WMF com `width != height` |
| `TestFERTFRecord_ADAR` | Garantir RTF decodificado sem vazar metadata |
| `TestFEChainNextCoords` | Validar `nextX/nextY` como coordenadas do proximo objeto |
| `TestFEEmptyTextNode_SLIM` | Documentar no vazio com `tln=0` |
| `TestFETruncatedEOF_ALGODAO` | Documentar comportamento para EOF truncado |
| `TestFEFalsePositiveInsideWMF_ADAR` | Garantir que falso-positivo nao gera elemento |
| `TestFENoDedupeByContent` | Garantir que objetos duplicados por texto/nome nao sejam descartados |

## Implicacoes para o parser

1. O mapeamento atual para texto esta conceitualmente correto, mas os nomes internos devem ser corrigidos.
2. O mapeamento atual para WMF bate com as amostras, mas precisa de testes numericos.
3. Remover dedupe por `text` e por `name` ou restringir a casos comprovadamente seguros.
4. Registrar nos testes os casos de falso-positivo, no vazio e EOF truncado.
