package docs

import "net/http"

const OpenAPI = `{
  "openapi": "3.0.3",
  "info": {
    "title": "Orca Slicer API",
    "version": "1.0.0",
    "description": "REST API em Go para gerenciar profiles JSON e executar slicing com OrcaSlicer."
  },
  "servers": [{ "url": "http://localhost:3000" }],
  "tags": [
    { "name": "Health", "description": "Healthcheck e disponibilidade" },
    { "name": "Profiles", "description": "CRUD/importação de profiles JSON" },
    { "name": "Aliases", "description": "Aliases para resolução de nomes de profiles" },
    { "name": "Resolve", "description": "Resolução de inherits, built-ins e overrides" },
    { "name": "Slicing", "description": "Slicing estrito de um STL, preview, status e debug" },
    { "name": "Jobs", "description": "Consulta, download e cancelamento de jobs assíncronos" },
    { "name": "Docs", "description": "OpenAPI e Swagger UI" }
  ],
  "paths": {
    "/health": {
      "get": {
        "tags": ["Health"],
        "summary": "Healthcheck completo",
        "responses": {
          "200": { "description": "Healthy", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/HealthResponse" } } } },
          "503": { "description": "Unhealthy", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/HealthResponse" } } } }
        }
      }
    },
    "/profile-aliases": {
      "get": {
        "tags": ["Aliases"],
        "summary": "Lista aliases de profiles",
        "responses": { "200": { "description": "Aliases" } }
      },
      "post": {
        "tags": ["Aliases"],
        "summary": "Cria ou atualiza alias de profile",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": { "schema": { "$ref": "#/components/schemas/ProfileAlias" } }
          }
        },
        "responses": { "200": { "description": "Aliases atualizados" } }
      }
    },
    "/profile-aliases/{category}/{from}": {
      "delete": {
        "tags": ["Aliases"],
        "summary": "Remove alias de profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "name": "from", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Aliases atualizados" }, "404": { "$ref": "#/components/responses/NotFound" } }
      }
    },
    "/profiles/{category}": {
      "get": {
        "tags": ["Profiles"],
        "summary": "Lista profiles com metadata",
        "parameters": [{ "$ref": "#/components/parameters/Category" }],
        "responses": { "200": { "description": "Profiles", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/ProfileListItem" } } } } } }
      }
    },
    "/profiles/{category}/{name}": {
      "get": {
        "tags": ["Profiles"],
        "summary": "Retorna JSON raw de um profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "200": { "description": "Profile JSON" }, "404": { "$ref": "#/components/responses/NotFound" } }
      },
      "delete": {
        "tags": ["Profiles"],
        "summary": "Remove profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "204": { "description": "Deleted" } }
      }
    },
    "/profiles/{category}/upload": {
      "post": {
        "tags": ["Profiles"],
        "summary": "Upload de profile JSON",
        "parameters": [{ "$ref": "#/components/parameters/Category" }],
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["name", "file"],
                "properties": {
                  "name": { "type": "string" },
                  "file": { "type": "string", "format": "binary" }
                }
              }
            }
          }
        },
        "responses": { "201": { "description": "Uploaded" } }
      }
    },
    "/profiles/{category}/import-url": {
      "post": {
        "tags": ["Profiles"],
        "summary": "Importa profile por URL HTTPS raw JSON",
        "parameters": [{ "$ref": "#/components/parameters/Category" }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/ImportRequest" }
            }
          }
        },
        "responses": { "201": { "description": "Imported" }, "409": { "$ref": "#/components/responses/Conflict" } }
      }
    },
    "/profiles/{category}/{name}/update-from-source": {
      "post": {
        "tags": ["Profiles"],
        "summary": "Atualiza profile usando sourceUrl salvo",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "200": { "description": "Updated or unchanged" } }
      }
    },
    "/profiles/resolve": {
      "post": {
        "tags": ["Resolve"],
        "summary": "Preview de um profile com overrides aplicados",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/ResolveProfileRequest" }
            }
          }
        },
        "responses": { "200": { "description": "Resolved profile" } }
      }
    },
    "/slice": {
      "post": {
        "tags": ["Slicing"],
        "summary": "Fatia exatamente um STL e retorna um G-code da mesa 1",
        "description": "Executa OrcaSlicer 2.5.0 com --slice 1, --clone-objects N e --export-statistics. Exige exatamente a quantidade solicitada em plate_1.gcode; nunca reduz quantity, entrega mesas adicionais, ZIP ou 3MF. Perfis e overrides são temporários.",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": { "$ref": "#/components/schemas/SliceMultipartRequest" }
            }
          }
        },
        "responses": {
          "200": { "$ref": "#/components/responses/GCodeSliceResult" },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "408": { "$ref": "#/components/responses/RequestTimeout" },
          "409": { "$ref": "#/components/responses/Conflict" },
          "422": { "$ref": "#/components/responses/UnprocessableSlice" },
          "500": { "$ref": "#/components/responses/InternalError" }
        }
      }
    },
    "/slice/preview": {
      "post": {
        "tags": ["Slicing"],
        "summary": "Fatia um STL e retorna toolpaths limitados, metadados e thumbnail",
        "description": "Usa o mesmo contrato estrito de quantidade e mesa do slicing síncrono. A prévia é extraída do G-code real, incluindo movimentos de suporte e brim; o thumbnail é uma renderização separada do STL. Não retorna o G-code. Fontes acima dos limites, comandos não suportados ou trabalho de arcos excessivo falham sem prévia parcial.",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": { "$ref": "#/components/schemas/SliceMultipartRequest" }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Preview gerado",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/PreviewResponse" } } }
          },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "408": { "$ref": "#/components/responses/RequestTimeout" },
          "409": { "$ref": "#/components/responses/Conflict" },
          "422": { "$ref": "#/components/responses/UnprocessableSlice" },
          "500": { "$ref": "#/components/responses/InternalError" }
        }
      }
    },
    "/slice/status": {
      "get": {
        "tags": ["Slicing"],
        "summary": "Status persistido do último slicing",
        "responses": { "200": { "description": "Status", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/JobState" } } } } }
      }
    },
    "/slice/debug": {
      "get": {
        "tags": ["Slicing"],
        "summary": "Debug persistido do último slicing",
        "responses": { "200": { "description": "Debug do último slicing", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/SliceDebug" } } } } }
      }
    },
    "/slice/async": {
      "post": {
        "tags": ["Slicing"],
        "summary": "Cria um job para fatiar exatamente um STL na mesa 1",
        "description": "Aceita o mesmo multipart de /slice, cria o job, inicia uma goroutine e responde 202 sem aguardar a conclusão do fatiamento; o fatiamento pode começar antes da resposta. Overflow, falta de evidências ou indisponibilidade do lock aparecem depois como job failed. O download de um job completed contém somente plate_1.gcode; 202 não reserva capacidade nem garante sucesso.",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": { "$ref": "#/components/schemas/SliceMultipartRequest" }
            }
          }
        },
        "responses": {
          "202": { "description": "Job criado; consulte /jobs/{id} até completed ou failed", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Job" } } } },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "500": { "$ref": "#/components/responses/InternalError" }
        }
      }
    },
    "/profiles/builtins/{category}": {
      "get": {
        "tags": ["Profiles"],
        "summary": "Lista profiles built-in do OrcaSlicer",
        "parameters": [{ "$ref": "#/components/parameters/Category" }],
        "responses": { "200": { "description": "Builtins", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/BuiltinProfile" } } } } } }
      }
    },
    "/jobs": {
      "get": {
        "tags": ["Jobs"],
        "summary": "Lista metadados dos jobs conhecidos",
        "description": "Sem paginação ou ordenação garantida; não inclui caminhos internos de resultados.",
        "responses": {
          "200": { "description": "Jobs conhecidos", "content": { "application/json": { "schema": { "type": "array", "items": { "$ref": "#/components/schemas/Job" } } } } }
        }
      }
    },
    "/jobs/{id}": {
      "get": {
        "tags": ["Jobs"],
        "summary": "Consulta status de um job assíncrono",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Job", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Job" } } } }, "404": { "$ref": "#/components/responses/NotFound" } }
      },
      "delete": {
        "tags": ["Jobs"],
        "summary": "Tenta cancelar um job conhecido",
        "description": "204 confirma apenas que a tentativa foi aceita para um job conhecido. Jobs já terminados e jobs ativos sem função de cancelamento registrada também podem retornar 204 sem alteração; não há estado cancelled nem garantia de interrupção imediata.",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "204": { "description": "Tentativa aceita; não comprova cancelamento" }, "404": { "$ref": "#/components/responses/NotFound" } }
      }
    },
    "/jobs/{id}/download": {
      "get": {
        "tags": ["Jobs"],
        "summary": "Baixa o único G-code validado de um job completed",
        "description": "Retorna application/x-gcode e plate_1.gcode. Um job failed por overflow ou pendente retorna 400, nunca um resultado parcial. Não inclui os headers de metadados do endpoint síncrono.",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "G-code completo da mesa 1", "content": { "application/x-gcode": { "schema": { "type": "string", "format": "binary" } } } },
          "400": { "$ref": "#/components/responses/BadRequest" },
          "404": { "$ref": "#/components/responses/NotFound" },
          "500": { "$ref": "#/components/responses/InternalError" }
        }
      }
    },
    "/slice/resolve-profiles": {
      "post": {
        "tags": ["Resolve"],
        "summary": "Preview dos profiles resolvidos para slicing",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "$ref": "#/components/schemas/SliceSettings" }
            }
          }
        },
        "responses": { "200": { "description": "Resolved profiles" } }
      }
    }
  },
  "components": {
    "responses": {
      "GCodeSliceResult": {
        "description": "G-code completo de plate_1.gcode, com todas as instâncias solicitadas",
        "headers": {
          "Content-Disposition": { "schema": { "type": "string" }, "example": "attachment; filename=plate_1.gcode" },
          "X-Print-Time-Seconds": { "description": "Estimativa em segundos; zero pode indicar metadado ausente", "schema": { "type": "number" } },
          "X-Filament-Used-g": { "description": "Massa estimada em gramas", "schema": { "type": "number" } },
          "X-Filament-Used-mm": { "description": "Comprimento estimado em milímetros", "schema": { "type": "number" } },
          "X-Slice-Quantity": { "description": "Quantidade validada nas estatísticas do OrcaSlicer", "schema": { "type": "integer", "minimum": 1, "maximum": 512 } },
          "X-Slice-Plate": { "description": "Sempre 1", "schema": { "type": "integer", "enum": [1] } }
        },
        "content": { "application/x-gcode": { "schema": { "type": "string", "format": "binary" } } }
      },
      "BadRequest": {
        "description": "Requisição inválida, incluindo múltiplos STLs, formato não STL ou combinação de parâmetros não suportada",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "quantity must be an integer between 1 and 512" } } }
      },
      "NotFound": {
        "description": "Recurso não encontrado",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "Profile not found" } } }
      },
      "RequestTimeout": {
        "description": "Slicing cancelado ou prazo do CLI excedido",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "Slicing cancelled or timed out" } } }
      },
      "Conflict": {
        "description": "Conflito ou slicer ocupado; não há fila de espera",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "Slicer is busy" } } }
      },
      "UnprocessableSlice": {
        "description": "Depois de o CLI concluir com sucesso, a validação detectou contagem ou mesas divergentes. Nenhum G-code parcial é retornado; reduza quantity e tente novamente. Falha do próprio CLI é 500; no async, ambos aparecem como job failed após o 202.",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "requested quantity 12, but OrcaSlicer produced 4 object(s) on plate 1; reduce quantity and retry" } } }
      },
      "InternalError": {
        "description": "Falha do CLI, filesystem, thumbnail, parser ou evidência de saída ausente/inválida",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "message": "Slicing failed" } } }
      }
    },
    "parameters": {
      "Category": {
        "name": "category",
        "in": "path",
        "required": true,
        "schema": { "type": "string", "enum": ["printers", "presets", "filaments"] }
      },
      "Name": {
        "name": "name",
        "in": "path",
        "required": true,
        "schema": { "type": "string" }
      }
    },
    "schemas": {
      "ErrorResponse": {
        "type": "object",
        "required": ["message"],
        "properties": {
          "message": { "type": "string" }
        }
      },
      "HealthResponse": {
        "type": "object",
        "example": {
          "status": "healthy",
          "dataPath": "/app/data",
          "dataPathExists": true,
          "dataPathWritable": true,
          "orcaSlicerPath": "/app/squashfs-root/AppRun",
          "orcaSlicerConfigured": true,
          "orcaSlicerExists": true,
          "orcaSlicerExecutable": true,
          "orcaProfilesPath": "/app/squashfs-root/resources/profiles",
          "orcaProfilesPathExists": true,
          "orcaVersion": "2.5.0",
          "generateImage": true,
          "resolveProfiles": true,
          "sanitizeProfiles": true
        },
        "properties": {
          "status": { "type": "string", "enum": ["healthy", "unhealthy"] },
          "dataPath": { "type": "string" },
          "dataPathExists": { "type": "boolean" },
          "dataPathWritable": { "type": "boolean" },
          "orcaSlicerPath": { "type": "string" },
          "orcaSlicerConfigured": { "type": "boolean" },
          "orcaSlicerExists": { "type": "boolean" },
          "orcaSlicerExecutable": { "type": "boolean" },
          "orcaProfilesPath": { "type": "string" },
          "orcaProfilesPathExists": { "type": "boolean" },
          "orcaVersion": { "type": "string" },
          "generateImage": { "type": "boolean" },
          "resolveProfiles": { "type": "boolean" },
          "sanitizeProfiles": { "type": "boolean" }
        }
      },
      "Metadata": {
        "type": "object",
        "properties": {
          "printTime": { "type": "number" },
          "filamentUsedG": { "type": "number" },
          "filamentUsedMm": { "type": "number" }
        }
      },
      "JobState": {
        "type": "object",
        "example": {
          "status": "completed",
          "startedAt": "2026-07-01T00:00:00Z",
          "finishedAt": "2026-07-01T00:02:00Z",
          "errorMessage": "",
          "files": ["/tmp/slice/output/plate_1.gcode"],
          "metadata": { "printTime": 12000, "filamentUsedG": 18.5, "filamentUsedMm": 6200 }
        },
        "properties": {
          "status": { "type": "string", "enum": ["idle", "processing", "completed", "failed", "cancelled"] },
          "startedAt": { "type": "string" },
          "finishedAt": { "type": "string" },
          "errorMessage": { "type": "string" },
          "files": { "type": "array", "items": { "type": "string" } },
          "metadata": { "$ref": "#/components/schemas/Metadata" }
        }
      },
      "SliceDebug": {
        "type": "object",
        "properties": {
          "startedAt": { "type": "string" },
          "finishedAt": { "type": "string" },
          "workdir": { "type": "string" },
          "inputPath": { "type": "string" },
          "outputDir": { "type": "string" },
          "command": { "type": "string" },
          "args": { "type": "array", "items": { "type": "string" } },
          "settings": { "type": "object" },
          "printer": { "type": "object" },
          "preset": { "type": "object" },
          "filament": { "type": "object" },
          "filaments": { "type": "array", "items": { "type": "object" } },
          "output": { "type": "string", "description": "stdout/stderr combinado, retido até 1 MiB com marcador de truncamento" },
          "resultJson": { "type": "object" },
          "files": { "type": "array", "items": { "type": "string" } },
          "slicerError": { "type": "string" },
          "errorMessage": { "type": "string" }
        }
      },
      "ProfileListItem": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "size": { "type": "integer" },
          "checksum": { "type": "string" },
          "updatedAt": { "type": "string" },
          "sourceUrl": { "type": "string" }
        }
      },
      "ProfileAlias": {
        "type": "object",
        "required": ["category", "from", "to"],
        "properties": {
          "category": { "type": "string", "enum": ["printers", "presets", "filaments"] },
          "from": { "type": "string" },
          "to": { "type": "string" }
        }
      },
      "ImportRequest": {
        "type": "object",
        "required": ["name", "url"],
        "properties": {
          "name": { "type": "string" },
          "url": { "type": "string", "format": "uri" },
          "overwrite": { "type": "boolean" }
        }
      },
      "ResolveProfileRequest": {
        "type": "object",
        "required": ["category", "name"],
        "properties": {
          "category": { "type": "string" },
          "name": { "type": "string" },
          "overrides": { "type": "object" }
        }
      },
      "SliceSettings": {
        "type": "object",
        "properties": {
          "printer": { "type": "string" },
          "preset": { "type": "string" },
          "filament": { "type": "string" },
          "overrides": { "type": "object" }
        }
      },
      "SliceMultipartRequest": {
        "type": "object",
        "description": "Exatamente um STL no campo file ou no alias files. Não são aceitos múltiplos modelos, STEP ou 3MF. Cada arquivo é limitado a 100000000 bytes, sem limite agregado da requisição.",
        "oneOf": [
          { "required": ["file"], "not": { "required": ["files"] } },
          { "required": ["files"], "not": { "required": ["file"] } }
        ],
        "properties": {
          "file": { "type": "string", "format": "binary", "description": "Único modelo STL; extensão sem distinção de caixa" },
          "files": { "type": "array", "minItems": 1, "maxItems": 1, "items": { "type": "string", "format": "binary" }, "description": "Alias legado para um único STL; não pode ser combinado com file" },
          "printer": { "type": "string", "description": "Nome salvo/vendor; ignorado quando printerProfile é enviado" },
          "preset": { "type": "string", "description": "Nome salvo/vendor; ignorado quando presetProfile é enviado" },
          "filament": { "type": "string", "description": "Nome salvo/vendor; ignorado quando filamentProfile é enviado" },
          "printerProfile": { "type": "string", "format": "binary", "description": "JSON direto completo/plano; tem precedência sobre printer" },
          "presetProfile": { "type": "string", "format": "binary", "description": "JSON direto completo/plano; tem precedência sobre preset" },
          "filamentProfile": { "type": "array", "items": { "type": "string", "format": "binary" }, "description": "Um ou mais JSON diretos; tem precedência sobre filament" },
          "quantity": { "type": "integer", "minimum": 1, "maximum": 512, "default": 1, "description": "Inclui o original; valores acima de 1 exigem arrange=true. Nunca é reduzido automaticamente" },
          "plate": { "type": "string", "enum": ["1"], "default": "1", "description": "Somente a mesa 1 é suportada" },
          "arrange": { "type": "boolean", "default": false, "description": "Obrigatório para quantity>1; o OrcaSlicer decide posição e espaçamento" },
          "arrangeSpacing": { "type": "number", "minimum": 0, "maximum": 0, "default": 0, "description": "0 usa espaçamento automático. Valores positivos retornam 400 porque o CLI 2.5.0 não oferece o caminho customizado suportado" },
          "orient": { "type": "boolean", "default": false, "description": "Orientação automática do OrcaSlicer; a presença de rotate, rotateX ou rotateY, mesmo com valor 0, é incompatível com orient=true" },
          "rotate": { "type": "number", "description": "Rotação explícita finita em graus ao redor de Z. Sua presença, inclusive com valor 0, exige orient=false e aciona --ensure-on-bed" },
          "rotateX": { "type": "number", "description": "Rotação explícita finita em graus ao redor de X. Sua presença, inclusive com valor 0, exige orient=false e aciona --ensure-on-bed" },
          "rotateY": { "type": "number", "description": "Rotação explícita finita em graus ao redor de Y. Sua presença, inclusive com valor 0, exige orient=false e aciona --ensure-on-bed" },
          "bedType": { "type": "string" },
          "exportType": { "type": "string", "enum": ["gcode"], "default": "gcode", "description": "3MF e ZIP não são suportados" },
          "multicolorOnePlate": { "type": "boolean", "default": false },
          "resolveProfiles": { "type": "boolean", "description": "Flag legada: perfis nomeados sempre têm inherits resolvido pela API; uploads diretos não" },
          "sanitizeProfiles": { "type": "boolean", "description": "Ajusta somente cópias temporárias: from=system e remoção de small_perimeter_speed em presets" },
          "generateImage": { "type": "boolean", "description": "Injeta thumbnail PNG 160x160 do STL no G-code sync/async. O preview gera thumbnail separado e força esta opção a false internamente" },
          "enableSupport": { "type": "boolean", "description": "Ausente preserva o preset; presente escreve enable_support como string 1/0 no preset temporário (necessário no OrcaSlicer 2.5.0)" },
          "brimType": { "type": "boolean", "description": "Ausente preserva o preset; true=auto_brim e false=no_brim" },
          "printSequenceByObject": { "type": "boolean", "default": false, "description": "true=by object; ausente/false=by layer no preset temporário" },
          "overrides": { "type": "string", "description": "String JSON com objetos printer, preset e/ou filament" }
        }
      },
      "PreviewResponse": {
        "type": "object",
        "required": ["usesSupport", "printTime", "filamentUsedG", "filamentUsedMm", "thumbnail", "quantity", "toolpaths"],
        "properties": {
          "usesSupport": { "type": "boolean", "description": "true quando toolpaths.supportSegments>0 no G-code completo analisado; não é garantia de imprimibilidade" },
          "printTime": { "type": "number", "description": "Estimativa em segundos" },
          "filamentUsedG": { "type": "number", "description": "Estimativa em gramas" },
          "filamentUsedMm": { "type": "number", "description": "Estimativa em milímetros" },
          "thumbnail": { "type": "string", "description": "PNG 160x160 em base64, sem prefixo data URI; renderização do STL original, não do toolpath ou da disposição final" },
          "quantity": { "type": "integer", "minimum": 1, "maximum": 512 },
          "toolpaths": { "$ref": "#/components/schemas/ToolpathPreview" }
        }
      },
      "ToolpathPreview": {
        "type": "object",
        "description": "Movimentos XY com extrusão positiva do G-code real. A leitura é completa somente quando sourceComplete=true; layers/segments são um prefixo retido e limitado, enquanto totais, contadores e bounds consideram toda a fonte analisada.",
        "required": ["version", "positionUnit", "extrusionUnit", "feedrateUnit", "coordinateSystem", "sourceComplete", "truncated", "sourceBytes", "sourceLines", "totalLayers", "totalSegments", "returnedLayers", "returnedSegments", "omittedSegments", "supportSegments", "brimSegments", "layerSemantics", "warnings", "limits", "layers"],
        "properties": {
          "version": { "type": "integer", "enum": [1] },
          "positionUnit": { "type": "string", "enum": ["mm"] },
          "extrusionUnit": { "type": "string", "enum": ["mm"] },
          "feedrateUnit": { "type": "string", "enum": ["mm/s"] },
          "coordinateSystem": { "type": "string", "enum": ["gcode"] },
          "sourceComplete": { "type": "boolean", "description": "true somente após leitura completa dentro dos limites de bytes, linhas e comandos suportados" },
          "truncated": { "type": "boolean", "description": "true quando camadas ou segmentos foram omitidos da representação retida; não significa leitura parcial da fonte" },
          "sourceBytes": { "type": "integer", "format": "int64", "minimum": 0 },
          "sourceLines": { "type": "integer", "minimum": 0 },
          "totalLayers": { "type": "integer", "minimum": 0 },
          "totalSegments": { "type": "integer", "minimum": 0, "description": "Total na fonte analisada após tesselação dos arcos" },
          "returnedLayers": { "type": "integer", "minimum": 0, "maximum": 512 },
          "returnedSegments": { "type": "integer", "minimum": 0, "maximum": 20000 },
          "omittedSegments": { "type": "integer", "minimum": 0 },
          "supportSegments": { "type": "integer", "minimum": 0, "description": "Total de extrusões Support, Support interface ou Support transition, incluindo segmentos omitidos" },
          "brimSegments": { "type": "integer", "minimum": 0, "description": "Total de extrusões Brim, incluindo segmentos omitidos" },
          "layerSemantics": { "type": "string", "enum": ["markers", "z_fallback"] },
          "warnings": { "type": "array", "items": { "type": "string" }, "description": "Inclui limitações como macros de impressora não expandidas e homing assumido em zero" },
          "limits": { "$ref": "#/components/schemas/ToolpathLimits" },
          "bounds": { "$ref": "#/components/schemas/ToolpathBounds" },
          "layers": { "type": "array", "maxItems": 512, "items": { "$ref": "#/components/schemas/ToolpathLayer" } }
        }
      },
      "ToolpathLimits": {
        "type": "object",
        "required": ["sourceBytes", "lineBytes", "lines", "layers", "segments", "arcSegmentsPerMove", "arcSegmentsTotal", "arcChordToleranceMm"],
        "properties": {
          "sourceBytes": { "type": "integer", "enum": [33554432] },
          "lineBytes": { "type": "integer", "enum": [1048576] },
          "lines": { "type": "integer", "enum": [4000000] },
          "layers": { "type": "integer", "enum": [512], "description": "Máximo retido; camadas adicionais são contadas, mas não retornadas" },
          "segments": { "type": "integer", "enum": [20000], "description": "Máximo retido globalmente" },
          "arcSegmentsPerMove": { "type": "integer", "enum": [4096] },
          "arcSegmentsTotal": { "type": "integer", "enum": [1000000], "description": "Orçamento global de tesselação; excedê-lo falha sem prévia parcial" },
          "arcChordToleranceMm": { "type": "number", "enum": [0.05] }
        }
      },
      "ToolpathBounds": {
        "type": "object",
        "required": ["min", "max"],
        "description": "Bounds dos pontos de extrusão XY analisados na fonte completa, inclusive geometria omitida da lista retida. Arcos são representados por sua tesselação.",
        "properties": {
          "min": { "$ref": "#/components/schemas/ToolpathVector" },
          "max": { "$ref": "#/components/schemas/ToolpathVector" }
        }
      },
      "ToolpathLayer": {
        "type": "object",
        "required": ["index", "z", "segments"],
        "properties": {
          "index": { "type": "integer", "minimum": 0 },
          "z": { "type": "number" },
          "segments": { "type": "array", "items": { "$ref": "#/components/schemas/ToolpathSegment" } }
        }
      },
      "ToolpathSegment": {
        "type": "object",
        "required": ["from", "to", "extrusion", "tool"],
        "properties": {
          "from": { "$ref": "#/components/schemas/ToolpathVector" },
          "to": { "$ref": "#/components/schemas/ToolpathVector" },
          "extrusion": { "type": "number", "description": "Delta positivo de filamento em mm atribuído ao movimento" },
          "feedrateMmSec": { "type": "number", "description": "Último feedrate comandado, convertido para mm/s" },
          "feature": { "type": "string", "description": "Marcador TYPE/FEATURE vigente; Wipe durante WIPE_START/WIPE_END" },
          "object": { "type": "string", "description": "Nome lido dos comentários printing object do OrcaSlicer" },
          "tool": { "type": "integer", "minimum": 0, "maximum": 254 }
        }
      },
      "ToolpathVector": {
        "type": "array",
        "minItems": 3,
        "maxItems": 3,
        "items": { "type": "number" },
        "description": "Coordenadas [X,Y,Z] em mm no sistema do G-code"
      },
      "Job": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "status": { "type": "string", "enum": ["pending", "slicing", "completed", "failed"] },
          "progress": { "type": "integer" },
          "error": { "type": "string" },
          "createdAt": { "type": "string", "format": "date-time" },
          "updatedAt": { "type": "string", "format": "date-time" }
        }
      },
      "BuiltinProfile": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "path": { "type": "string" }
        }
      }
    }
  }
}`

func OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(OpenAPI))
}

func SwaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
  <title>Orca Slicer API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>SwaggerUIBundle({ url: '/openapi.json', dom_id: '#swagger-ui' });</script>
</body>
</html>`))
}
