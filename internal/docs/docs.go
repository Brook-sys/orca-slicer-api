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
    { "name": "Slicing", "description": "Slicing, preview, status e debug" },
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
        "summary": "Executa slicing síncrono",
        "description": "Executa OrcaSlicer CLI com profiles por nome ou JSON multipart. Aplica overrides temporários como enableSupport, brimType e printSequenceByObject sem modificar profiles salvos.",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "required": ["file"],
                "properties": {
                  "file": { "type": "string", "format": "binary" },
                  "printerProfile": { "type": "string", "format": "binary", "description": "Profile JSON cru da impressora. Se enviado, tem prioridade sobre printer." },
                  "presetProfile": { "type": "string", "format": "binary", "description": "Profile JSON cru do process/preset. Se enviado, tem prioridade sobre preset." },
                  "filamentProfile": { "type": "array", "items": { "type": "string", "format": "binary" }, "description": "Um ou mais profile JSON crus de filamento. Se enviado, tem prioridade sobre filament." },
                  "printer": { "type": "string", "description": "Nome de profile salvo em /profiles/printers. Ignorado quando printerProfile é enviado." },
                  "preset": { "type": "string", "description": "Nome de profile salvo em /profiles/presets. Ignorado quando presetProfile é enviado." },
                  "filament": { "type": "string", "description": "Nome de profile salvo em /profiles/filaments. Ignorado quando filamentProfile é enviado." },
                  "bedType": { "type": "string" },
                  "plate": { "type": "string" },
                  "arrange": { "type": "boolean" },
                  "orient": { "type": "boolean" },
                  "exportType": { "type": "string", "enum": ["gcode", "3mf"] },
                  "multicolorOnePlate": { "type": "boolean" },
                  "resolveProfiles": { "type": "boolean", "description": "Quando true, resolve inherits/built-ins para profiles selecionados por nome antes do slicing. Não afeta printerProfile/presetProfile/filamentProfile enviados como arquivo." },
                  "sanitizeProfiles": { "type": "boolean", "description": "Quando true, ajusta campos conhecidos por quebrar o Orca CLI apenas nos profiles temporários: define from=system em todos os profiles e remove small_perimeter_speed em presets." },
                  "generateImage": { "type": "boolean", "description": "Quando true, gera thumbnail PNG 160x160 do modelo STL e injeta no G-code no formato esperado pelo Elegoo Neptune 4. Modelo em #22948a, fundo transparente." },
                  "enableSupport": { "type": "boolean", "description": "Quando definido (true/false), injeta enable_support no preset temporário. Sobrescreve o valor do preset salvo. Permite ativar/desativar suportes por chamada sem alterar o profile salvo." },
                  "brimType": { "type": "boolean", "description": "Quando definido (true/false), injeta brim_type no preset temporário: true=auto_brim, false=no_brim. Permite controlar brim por chamada sem alterar o profile salvo." },
                  "printSequenceByObject": { "type": "boolean", "description": "Quando true, injeta print_sequence=by object no preset temporário. Quando false (padrão), injeta print_sequence=by layer. Permite controlar sequência de impressão por chamada sem alterar o profile salvo." },
                  "overrides": { "type": "string", "description": "JSON string com overrides por printer/preset/filament" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Arquivo gerado" }, "400": { "$ref": "#/components/responses/BadRequest" }, "409": { "$ref": "#/components/responses/Conflict" }, "500": { "$ref": "#/components/responses/InternalError" } }
      }
    },
    "/slice/preview": {
      "post": {
        "tags": ["Slicing"],
        "summary": "Gera preview JSON com thumbnail e detecção de suporte",
        "description": "Executa slicing forçando enableSupport=true, detecta suporte no G-code e retorna JSON com usesSupport, metadata e thumbnail base64 PNG. Não retorna o G-code binário.",
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
          "409": { "$ref": "#/components/responses/Conflict" },
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
        "tags": ["Slice"],
        "summary": "Inicia slicing assíncrono com suporte a múltiplos arquivos",
        "requestBody": {
          "required": true,
          "content": {
            "multipart/form-data": {
              "schema": {
                "type": "object",
                "properties": {
                  "file": { "type": "string", "format": "binary" },
                  "files": { "type": "array", "items": { "type": "string", "format": "binary" } },
                  "printer": { "type": "string" },
                  "preset": { "type": "string" },
                  "filament": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": { "202": { "description": "Job criado", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Job" } } } } }
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
    "/jobs/{id}": {
      "get": {
        "tags": ["Jobs"],
        "summary": "Consulta status de um job assíncrono",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Job", "content": { "application/json": { "schema": { "$ref": "#/components/schemas/Job" } } } }, "404": { "$ref": "#/components/responses/NotFound" } }
      },
      "delete": {
        "tags": ["Jobs"],
        "summary": "Cancela job em execução",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "204": { "description": "Cancelled" }, "404": { "$ref": "#/components/responses/NotFound" } }
      }
    },
    "/jobs/{id}/download": {
      "get": {
        "tags": ["Jobs"],
        "summary": "Download do resultado quando status=completed",
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Arquivo" }, "400": { "description": "Não pronto" }, "404": { "$ref": "#/components/responses/NotFound" } }
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
      "BadRequest": {
        "description": "Requisição inválida",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "error": "Invalid multipart form" } } }
      },
      "NotFound": {
        "description": "Recurso não encontrado",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "error": "Profile not found" } } }
      },
      "Conflict": {
        "description": "Conflito ou slicer ocupado",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "error": "Slicer is busy" } } }
      },
      "InternalError": {
        "description": "Erro interno",
        "content": { "application/json": { "schema": { "$ref": "#/components/schemas/ErrorResponse" }, "example": { "error": "Slicing failed" } } }
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
        "properties": {
          "error": { "type": "string" }
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
          "orcaVersion": "2.4.1",
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
          "files": ["/tmp/slice/output/result.gcode"],
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
          "output": { "type": "string" },
          "resultJSON": { "type": "object" },
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
        "required": ["file"],
        "properties": {
          "file": { "type": "string", "format": "binary" },
          "printer": { "type": "string" },
          "preset": { "type": "string" },
          "filament": { "type": "string" },
          "printerProfile": { "type": "string", "format": "binary" },
          "presetProfile": { "type": "string", "format": "binary" },
          "filamentProfile": { "type": "array", "items": { "type": "string", "format": "binary" } },
          "resolveProfiles": { "type": "boolean" },
          "sanitizeProfiles": { "type": "boolean" },
          "generateImage": { "type": "boolean" },
          "enableSupport": { "type": "boolean" },
          "brimType": { "type": "boolean" },
          "printSequenceByObject": { "type": "boolean" },
          "overrides": { "type": "string" }
        }
      },
      "PreviewResponse": {
        "type": "object",
        "example": {
          "usesSupport": true,
          "printTime": 12450.3,
          "filamentUsedG": 18.7,
          "filamentUsedMm": 12450.3,
          "thumbnail": "iVBORw0KGgoAAAANSUhEUgAAAKAAAACgCAYAAACLz2ct..."
        },
        "properties": {
          "usesSupport": { "type": "boolean" },
          "printTime": { "type": "number" },
          "filamentUsedG": { "type": "number" },
          "filamentUsedMm": { "type": "number" },
          "thumbnail": { "type": "string", "description": "PNG 160x160 em base64" }
        }
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
