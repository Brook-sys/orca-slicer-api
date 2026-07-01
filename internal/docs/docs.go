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
  "paths": {
    "/health": {
      "get": {
        "summary": "Healthcheck completo",
        "responses": { "200": { "description": "Healthy" }, "503": { "description": "Unhealthy" } }
      }
    },
    "/profile-aliases": {
      "get": {
        "summary": "Lista aliases de profiles",
        "responses": { "200": { "description": "Aliases" } }
      },
      "post": {
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
        "summary": "Remove alias de profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "name": "from", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Aliases atualizados" }, "404": { "description": "Alias não encontrado" } }
      }
    },
    "/profiles/{category}": {
      "get": {
        "summary": "Lista profiles com metadata",
        "parameters": [{ "$ref": "#/components/parameters/Category" }],
        "responses": { "200": { "description": "Profiles" } }
      }
    },
    "/profiles/{category}/{name}": {
      "get": {
        "summary": "Retorna JSON raw de um profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "200": { "description": "Profile JSON" }, "404": { "description": "Not found" } }
      },
      "delete": {
        "summary": "Remove profile",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "204": { "description": "Deleted" } }
      }
    },
    "/profiles/{category}/upload": {
      "post": {
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
        "responses": { "201": { "description": "Imported" }, "409": { "description": "Already exists" } }
      }
    },
    "/profiles/{category}/{name}/update-from-source": {
      "post": {
        "summary": "Atualiza profile usando sourceUrl salvo",
        "parameters": [{ "$ref": "#/components/parameters/Category" }, { "$ref": "#/components/parameters/Name" }],
        "responses": { "200": { "description": "Updated or unchanged" } }
      }
    },
    "/profiles/resolve": {
      "post": {
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
        "summary": "Executa slicing síncrono",
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
                  "overrides": { "type": "string", "description": "JSON string com overrides por printer/preset/filament" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Arquivo gerado" }, "409": { "description": "Slicer busy" } }
      }
    },
    "/slice/status": {
      "get": {
        "summary": "Status persistido do último slicing",
        "responses": { "200": { "description": "Status" } }
      }
    },
    "/slice/debug": {
      "get": {
        "summary": "Debug persistido do último slicing",
        "responses": { "200": { "description": "Debug do último slicing" } }
      }
    },
    "/slice/resolve-profiles": {
      "post": {
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
