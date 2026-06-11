# Grok Research: Delegation Round 2

Fonte: seis sessoes `grok` em modo somente leitura, sem limite de turnos.

## Achados principais

| Frente | Resultado |
|---|---|
| Serializacao base | Funcao exata do blob `FE FF FF FF` nao foi localizada; decompile confirma runtime e registry `0xFE`, ETQ real confirma layout binario |
| Alinhamento texto | `+0x42` CadMapa aparece no stream texto em `FE+32..+33` (`int16` LE); corpus atual usa `0` em todos os textos |
| Parser/model | Remover dedupe por conteudo; style e bitmask; nomes `TextW/TextH` estavam semanticamente invertidos |
| Testes | Suite recomendada com offsets fixos para texto, WMF, RTF, linked-list, falso positivo, no vazio e EOF truncado |
| Render/impressao | Canvas e impressao devem usar retangulo externo; fonte efetiva vem da altura do `RECT` |
| Toolbar dropdown | Dropdown estilo Win32 e viavel com `walk.NewMenu` + `win.TrackPopupMenuEx` ancorado no rect do combo pintado |

## Decisoes de implementacao imediata

1. Parsear alinhamento texto em `FE+32` e propagar como `left`, `center`, `right`.
2. Nao tratar ainda o byte pos-registro ETQ (`4/5/6`) como bitmask GDI; o bitmask e confirmado para o descritor de fonte CadMapa, mas a equivalencia desse byte do stream ainda nao esta provada.
3. Remover dedupe por texto e por nome WMF.
4. Renomear campos internos de texto para `RectHeight` e `RectWidth`.
5. Manter WMF usando rect externo confirmado por ETQ: `head=width/height`, pre-block=`x/y`.
6. Atualizar canvas para usar altura do retangulo como tamanho efetivo de fonte quando disponivel.

## Pendencias para nova rodada

1. RTF/RECT: o registro ADAR `FEITO NO BRASIL` tem `post1=45`, `post2=562` e `\fs16`; o caso e atipico e nao deve virar heuristica visual sem mais amostras/decompile do loader base.
2. Implementar dropdowns da toolbar via `TrackPopupMenuEx`.
3. Sidecar/layout fallback: evitar mascarar parser ETQ em modo de validacao.
