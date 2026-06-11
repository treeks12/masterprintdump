# Grok Research: CadMapa WMF

Fonte: pesquisa delegada ao `grok` em modo somente leitura, sem `--max-turns`.

## Funcoes analisadas

| Funcao | Papel |
|---|---|
| `FUN_00423530` | Render WMF/EMF via `PlayEnhMetaFile(hdc, hmf, rect)` |
| `FUN_00423740` | Getter de largura intrinseca do metafile |
| `FUN_00423728` | Getter de altura intrinseca do metafile |
| `FUN_00423d68` | Setter de largura intrinseca |
| `FUN_00423d2c` | Setter de altura intrinseca |
| `FUN_004238c4` | Load do payload metafile |
| `FUN_00423f00` | Save do payload metafile |
| `FUN_00423a00` | Load Aldus WMF |
| `FUN_00423930` | Load EMF |
| `FUN_00424010` | Save Aldus WMF |
| `FUN_00423f84` | Save EMF |

## Ordem de retangulo em memoria

`FUN_004391d8` confirma o objeto grafico com:

| Offset | Campo |
|---|---|
| `+0x30` | X / left |
| `+0x34` | Y / top |
| `+0x38` | Width |
| `+0x3c` | Height |

O retangulo de paint e montado como `left=x`, `top=y`, `right=x+width`, `bottom=y+height`.

## Payload Aldus/EMF versus retangulo externo

| Camada | Papel |
|---|---|
| Header Aldus/EMF | Dimensoes intrinsecas do metafile |
| Retangulo externo do objeto | Posicao e tamanho do objeto na etiqueta |
| `PlayEnhMetaFile` | Estica/desenha o metafile no retangulo externo |

Conclusao: nao usar bounds internos do Aldus como posicao/tamanho de colocacao. O retangulo externo do objeto e a fonte de verdade para render.

## Campos confirmados

| Campo | Evidencia |
|---|---|
| Assinatura Aldus | `0x9AC6CDD7` |
| Assinatura EMF | `0x464D4520` |
| Conversao centi-mm | fator `2540` |
| Formato do objeto | `x,y,width,height` em `+0x30..+0x3c` |
| Render | `PlayEnhMetaFile` recebe retangulo externo |
| Tipo WMF runtime | flag `0x80000008` aparece no caminho de tipo |

## Hipoteses ainda nao provadas no decompile

1. O mapeamento byte-a-byte do registro ETQ `FE FF FF FF` nao aparece diretamente nas funcoes `00423xxx`.
2. O parser atual usa `preOff = i - 83` para pegar `x,y` de WMF; isso esta suportado por amostras ETQ, mas ainda deve ser documentado como layout de stream, nao como payload WMF.
3. `embeddedWMFBySize` identifica simbolos por tamanho; util, mas menos robusto que hash/checksum.

## Recomendacoes

1. Separar claramente parser de payload WMF e parser de retangulo externo.
2. Manter `width=head0`, `height=head1`, `x=post1`, `y=post2` para WMF apenas enquanto validado por amostras ETQ.
3. Nao usar dimensoes Aldus como rect de colocacao.
4. Validar Aldus com checksum XOR quando possivel.
5. No canvas e impressao, sempre desenhar WMF no rect externo decodificado.
