package ai

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/thomas-vilte/matecommit/internal/models"
)

const (
	issueReferenceInstructionsES = `Si hay un issue asociado (#{{.IssueNumber}}), DEBES incluir la referencia en el título del commit:
       - Para features/mejoras: "tipo: mensaje (#{{.IssueNumber}})"
       - Para bugs: "fix: mensaje (#{{.IssueNumber}})" o "fix(scope): mensaje (fixes #{{.IssueNumber}})"
       - Ejemplos válidos:
         ✅ feat: add dark mode support (#{{.IssueNumber}})
         ✅ fix: resolve authentication error (fixes #{{.IssueNumber}})
         ✅ feat(api): implement caching layer (#{{.IssueNumber}})
       - No omitas la referencia del issue #{{.IssueNumber}}.`

	issueReferenceInstructionsEN = `There is an associated issue (#{{.IssueNumber}}), you MUST include the reference in the commit title:
       - For features/improvements: "type: message (#{{.IssueNumber}})"
       - For bugs: "fix: message (#{{.IssueNumber}})" or "fix(scope): message (fixes #{{.IssueNumber}})"
       - Valid examples:
         ✅ feat: add dark mode support (#{{.IssueNumber}})
         ✅ fix: resolve authentication error (fixes #{{.IssueNumber}})
         ✅ feat(api): implement caching layer (#{{.IssueNumber}})
       - NEVER omit the reference to issue #{{.IssueNumber}}.`
)

// PromptData holds the parameters for template rendering
type PromptData struct {
	Count           int
	Files           string
	Diff            string
	Ticket          string
	History         string
	Instructions    string
	IssueNumber     int
	RelatedIssues   string
	IssueInfo       string
	RepoOwner       string
	RepoName        string
	PreviousVersion string
	CurrentVersion  string
	LatestVersion   string
	ReleaseDate     string
	Changelog       string
	PRContent       string
	TechnicalInfo   string
}

// RenderPrompt renders a prompt template with the provided data
func RenderPrompt(name, tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", name, err)
	}

	return buf.String(), nil
}

const (
	prPromptTemplateEN = `# Task
  Act as a Senior Tech Lead and generate a Pull Request summary.

  # PR Content
  {{.PRContent}}

  # Golden Rules (Constraints)
  1. **No Hallucinations:** If it's not in the diff, DO NOT invent it.
  2. **Tone:** Professional, direct, technical. Use first person ("I implemented", "I added").
  3. **Format:** Raw JSON only. Do not wrap in markdown blocks (like ` + "" + `).

  # Instructions
  1. Title: Catchy but descriptive (max 80 chars).
  2. Key Changes: Filter the noise. Explain the *technical impact*, not just the code change.
  3. Labels: Choose wisely (feature, fix, refactor, docs, infra, test, breaking-change).

  # STRICT OUTPUT FORMAT
  ⚠️ CRITICAL: You MUST return ONLY valid JSON. No markdown blocks, no explanations, no text before/after.
  ⚠️ ALL field types are STRICTLY enforced. DO NOT change types or add extra fields.
  
  ## JSON Schema (MANDATORY):
  {
    "type": "object",
    "required": ["title", "body", "labels"],
    "properties": {
      "title": {
        "type": "string",
        "description": "PR title (max 80 chars)"
      },
      "body": {
        "type": "string",
        "description": "Detailed markdown body with overview, key changes, and technical impact"
      },
      "labels": {
        "type": "array",
        "items": {
          "type": "string"
        },
        "description": "Array of label strings (feature, fix, refactor, docs, infra, test, breaking-change)"
      }
    },
    "additionalProperties": false
  }

  ## Type Rules (STRICT):
  - "title": MUST be string (never number, never null, never empty)
  - "body": MUST be string (never number, never null, can contain markdown)
  - "labels": MUST be array of strings (never array of numbers, never null, use [] if empty)

  ## Prohibited Actions:
  ❌ DO NOT add any fields not listed in the schema
  ❌ DO NOT change field types (e.g., title to number)
  ❌ DO NOT wrap JSON in markdown code blocks
  ❌ DO NOT add explanatory text before/after JSON
  ❌ DO NOT use null values for required fields

  ## Valid Example:
  {
    "title": "feat(auth): implement OAuth2 authentication",
    "body": "## Overview\nI implemented OAuth2 authentication to improve security.\n\n## Key Changes\n- Added OAuth2 client\n- Updated login flow\n\n## Technical Impact\nImproves security and allows SSO integration.",
    "labels": ["feature", "auth"]
  }

  Generate the summary now. Return ONLY the JSON object, nothing else.`

	prPromptTemplateES = `# Tarea
  Actuá como un Desarrollador Senior y genera un resumen del Pull Request.

  # Contenido del PR
  {{.PRContent}}

  # Reglas de Oro (Constraints)
  1. **Cero alucinaciones:** Si algo no está explícito en el diff, no lo inventes.
  2. **Tono:** Profesional, cercano y directo. Usa primera persona ("Implementé", "Agregué", "Corregí"). Evita el lenguaje robótico ("Se ha realizado").
  3. **Formato:** JSON crudo. No incluyas bloques de markdown.

  # Instrucciones
  1. Título: Descriptivo y conciso (máx 80 caracteres).
  2. Cambios Clave: Filtrá el ruido. Explicá el *impacto* técnico y el propósito, no solo qué línea cambió.
  3. Etiquetas: Elegí con criterio (feature, fix, refactor, docs, infra, test, breaking-change).

  # FORMATO DE SALIDA ESTRICTO
  ⚠️ CRÍTICO: DEBES devolver SOLO JSON válido. Sin bloques de markdown, sin explicaciones, sin texto antes/después.
  ⚠️ TODOS los tipos de campos están ESTRICTAMENTE definidos. NO cambies tipos ni agregues campos extra.
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.
  
  ## Schema JSON (OBLIGATORIO):
  {
    "type": "object",
    "required": ["title", "body", "labels"],
    "properties": {
      "title": {
        "type": "string",
        "description": "Título del PR (máx 80 caracteres)"
      },
      "body": {
        "type": "string",
        "description": "Cuerpo detallado en markdown con resumen, cambios clave e impacto técnico"
      },
      "labels": {
        "type": "array",
        "items": {
          "type": "string"
        },
        "description": "Array de etiquetas como strings (feature, fix, refactor, docs, infra, test, breaking-change)"
      }
    },
    "additionalProperties": false
  }

  ## Reglas de Tipos (ESTRICTAS):
  - "title": DEBE ser string (nunca número, nunca null, nunca vacío)
  - "body": DEBE ser string (nunca número, nunca null, puede contener markdown)
  - "labels": DEBE ser array de strings (nunca array de números, nunca null, usar [] si está vacío)

  ## Acciones Prohibidas:
  ❌ NO agregues campos que no estén en el schema
  ❌ NO cambies tipos de campos (ej: title a número)
  ❌ NO envuelvas el JSON en bloques de markdown
  ❌ NO agregues texto explicativo antes/después del JSON
  ❌ NO uses null para campos requeridos

  ## Ejemplo Válido:
  {
    "title": "feat(auth): implementar autenticación OAuth2",
    "body": "## Resumen\nImplementé autenticación OAuth2 para mejorar la seguridad.\n\n## Cambios Clave\n- Agregué cliente OAuth2\n- Actualicé el flujo de login\n\n## Impacto Técnico\nMejora la seguridad y permite integración SSO.",
    "labels": ["feature", "auth"]
  }

  Genera el resumen ahora. Devuelve SOLO el objeto JSON, nada más.`
)

const (
	promptTemplateWithTicketEN = `# Task
  Act as a Git Specialist and generate {{.Count}} commit message suggestions.

  # Context
  - Modified Files: {{.Files}}
  - Diff: {{.Diff}}
  - Ticket/Issue: {{.Ticket}}
  - Recent History: {{.History}}
  - Issue Instructions: {{.Instructions}}

  # Quality Guidelines
  1. **Conventional Commits:** Strictly follow ` + "`type(scope): description`" + `.
     - Types: feat, fix, refactor, perf, test, docs, chore, build, ci.
  2. **Precision:**
     - ❌ BAD: "fix: various fixes in login" (Too vague)
     - ✅ GOOD: "fix(auth): handle null token error (#42)" (Precise)
  3. **Scope:** If you touched 'ui' files, scope is (ui). If 'api', then (api).
  4. **Style:**
     - Title: Imperative mood ("add", not "added").
     - Description: First person, professional tone ("I optimized the query...").
  5. **Requirements Validation (IMPORTANT):**
     - Analyze ONLY the current diff changes against ticket criteria.
     - Mark as "missing" ONLY requirements that are NOT visible in the diff.
     - If recent history shows something was implemented in previous commits, do NOT mark it as missing.
     - If you see file names or function names in the diff indicating prior implementation (e.g., "stats.go", "CountTokens"), assume it exists.
     - Focus on what's missing NOW in the current commit context, not in the entire project.

  # STRICT OUTPUT FORMAT
  ⚠️ CRITICAL: You MUST return ONLY valid JSON. No markdown blocks, no explanations, no text before/after.
  ⚠️ ALL field types are STRICTLY enforced. DO NOT change types or add extra fields.
  
  ## JSON Schema (MANDATORY):
  {
    "type": "array",
    "items": {
      "type": "object",
      "required": ["title", "desc", "files"],
      "properties": {
        "title": {
          "type": "string",
          "description": "Commit title (type(scope): message)"
        },
        "desc": {
          "type": "string",
          "description": "Detailed explanation in first person"
        },
        "files": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Array of file paths as strings"
        },
        "analysis": {
          "type": "object",
          "required": ["overview", "purpose", "impact"],
          "properties": {
            "overview": {"type": "string"},
            "purpose": {"type": "string"},
            "impact": {"type": "string"}
          },
          "additionalProperties": false
        },
        "requirements": {
          "type": "object",
          "required": ["status", "missing", "completed_indices", "suggestions"],
          "properties": {
            "status": {
              "type": "string",
              "enum": ["full_met", "partially_met", "not_met"]
            },
            "missing": {
              "type": "array",
              "items": {"type": "string"}
            },
            "completed_indices": {
              "type": "array",
              "items": {"type": "integer"}
            },
            "suggestions": {
              "type": "array",
              "items": {"type": "string"}
            }
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    }
  }

  ## Type Rules (STRICT):
  - "title": MUST be string (never number, never null)
  - "desc": MUST be string (never number, never null, can be empty string "")
  - "files": MUST be array of strings (never array of numbers, never null)
  - "analysis.overview": MUST be string
  - "analysis.purpose": MUST be string
  - "analysis.impact": MUST be string
  - "requirements.status": MUST be one of: "full_met" | "partially_met" | "not_met" (exact strings)
  - "requirements.missing": MUST be array of strings (never null, use [] if empty)
  - "requirements.completed_indices": MUST be array of integers (never strings, never null, use [] if empty)
  - "requirements.suggestions": MUST be array of strings (never null, use [] if empty)

  ## Prohibited Actions:
  ❌ DO NOT add any fields not listed in the schema
  ❌ DO NOT change field types (e.g., desc to number)
  ❌ DO NOT wrap JSON in markdown code blocks
  ❌ DO NOT add explanatory text before/after JSON
  ❌ DO NOT use null values for required string fields (use "" instead)

  ## Valid Example:
  [
    {
      "title": "fix(auth): handle null token error (#42)",
      "desc": "I added validation to prevent null token errors in the authentication flow",
      "files": ["internal/auth/auth.go", "internal/auth/auth_test.go"],
      "analysis": {
        "overview": "Added null check for token",
        "purpose": "Prevent panic when token is null",
        "impact": "Improves error handling"
      },
      "requirements": {
        "status": "full_met",
        "missing": [],
        "completed_indices": [0, 1],
        "suggestions": []
      }
    }
  ]

  Generate {{.Count}} suggestions now. Return ONLY the JSON array, nothing else.`

	promptTemplateWithTicketES = `# Tarea
  Actuá como un especialista en Git y genera {{.Count}} sugerencias de commits.
  
  # Contexto
  - Archivos: {{.Files}}
  - Diff: {{.Diff}}
  - Ticket/Issue: {{.Ticket}}
  - Historial reciente: {{.History}}
  - Instrucciones Issue: {{.Instructions}}

  # Criterios de Calidad (Guidelines)
  1. **Conventional Commits:** Respeta estrictamente ` + "`tipo(scope): descripción`" + `.
     - Tipos: feat, fix, refactor, perf, test, docs, chore, build, ci.
  2. **Precisión:**
     - ❌ MAL: "fix: arreglos varios en el login" (Muy vago)
     - ✅ BIEN: "fix(auth): manejo de error en token nulo (#42)" (Preciso)
  3. **Scope:** Si tocaste archivos de 'ui', el scope es (ui). Si es 'api', es (api). Si son muchos, no uses scope.
  4. **Primera Persona:** La descripción ("desc") escribila como si le contaras a un colega (ej: "Optimicé la query para mejorar el tiempo de respuesta").
  5. **Validación de Requerimientos (IMPORTANTE):**
     - Analiza SOLO los cambios del diff actual contra los criterios del ticket.
     - Marca como "missing" ÚNICAMENTE requisitos que NO están visibles en el diff.
     - Si el historial reciente muestra que algo ya se implementó en commits anteriores, NO lo marques como faltante.
     - Si ves nombres de archivos o funciones en el diff que indican implementación previa (ej: "stats.go", "CountTokens"), asume que ya existe.
     - Enfocate en lo que falta AHORA en el contexto del commit actual, no en el proyecto completo.

  # FORMATO DE SALIDA ESTRICTO
  ⚠️ CRÍTICO: DEBES devolver SOLO JSON válido. Sin bloques de markdown, sin explicaciones, sin texto antes/después.
  ⚠️ TODOS los tipos de campos están ESTRICTAMENTE definidos. NO cambies tipos ni agregues campos extra.
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.
  
  ## Schema JSON (OBLIGATORIO):
  {
    "type": "array",
    "items": {
      "type": "object",
      "required": ["title", "desc", "files"],
      "properties": {
        "title": {
          "type": "string",
          "description": "Título del commit (tipo(scope): mensaje)"
        },
        "desc": {
          "type": "string",
          "description": "Explicación detallada en primera persona"
        },
        "files": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Array de rutas de archivos como strings"
        },
        "analysis": {
          "type": "object",
          "required": ["overview", "purpose", "impact"],
          "properties": {
            "overview": {"type": "string"},
            "purpose": {"type": "string"},
            "impact": {"type": "string"}
          },
          "additionalProperties": false
        },
        "requirements": {
          "type": "object",
          "required": ["status", "missing", "completed_indices", "suggestions"],
          "properties": {
            "status": {
              "type": "string",
              "enum": ["full_met", "partially_met", "not_met"]
            },
            "missing": {
              "type": "array",
              "items": {"type": "string"}
            },
            "completed_indices": {
              "type": "array",
              "items": {"type": "integer"}
            },
            "suggestions": {
              "type": "array",
              "items": {"type": "string"}
            }
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    }
  }

  ## Reglas de Tipos (ESTRICTAS):
  - "title": DEBE ser string (nunca número, nunca null)
  - "desc": DEBE ser string (nunca número, nunca null, puede ser "" si está vacío)
  - "files": DEBE ser array de strings (nunca array de números, nunca null)
  - "analysis.overview": DEBE ser string
  - "analysis.purpose": DEBE ser string
  - "analysis.impact": DEBE ser string
  - "requirements.status": DEBE ser uno de: "full_met" | "partially_met" | "not_met" (strings exactos)
  - "requirements.missing": DEBE ser array de strings (nunca null, usar [] si está vacío)
  - "requirements.completed_indices": DEBE ser array de enteros (nunca strings, nunca null, usar [] si está vacío)
  - "requirements.suggestions": DEBE ser array de strings (nunca null, usar [] si está vacío)

  ## Acciones Prohibidas:
  ❌ NO agregues campos que no estén en el schema
  ❌ NO cambies tipos de campos (ej: desc a número)
  ❌ NO envuelvas el JSON en bloques de markdown
  ❌ NO agregues texto explicativo antes/después del JSON
  ❌ NO uses null para campos string requeridos (usa "" en su lugar)

  ## Ejemplo Válido:
  [
    {
      "title": "fix(auth): manejo de error en token nulo (#42)",
      "desc": "Agregué validación para prevenir errores cuando el token es nulo",
      "files": ["internal/auth/auth.go", "internal/auth/auth_test.go"],
      "analysis": {
        "overview": "Agregué validación de token nulo",
        "purpose": "Prevenir panic cuando el token es null",
        "impact": "Mejora el manejo de errores"
      },
      "requirements": {
        "status": "full_met",
        "missing": [],
        "completed_indices": [0, 1],
        "suggestions": []
      }
    }
  ]

  Genera {{.Count}} sugerencias ahora. Devuelve SOLO el array JSON, nada más.`
)

const (
	promptTemplateWithoutTicketES = `# Tarea
  Actuá como un especialista en Git y genera {{.Count}} sugerencias de commits basadas en el código.

  # Inputs
  - Archivos Modificados: {{.Files}}
  - Cambios (Diff): {{.Diff}}
  - Instrucciones Issues: {{.Instructions}}
  - Historial: {{.History}}

  # Estrategia de Generación
  1. **Analiza el Diff:** Identifica qué lógica cambió realmente. Ignora cambios de formato/espacios.
  2. **Categoriza:**
     - ¿Nueva feature? -> feat
     - ¿Arreglo de bug? -> fix
     - ¿Cambio de código sin cambio de lógica? -> refactor
     - ¿Solo documentación? -> docs
  3. **Redacta:**
     - Título: Imperativo, max 50 chars si es posible (ej: "agrega validación", no "agregando").
     - Descripción: Primera persona, tono profesional y natural. "Agregué esta validación para evitar X error".

  # Ejemplos de Estilo
  - ❌ "update main.go" (Pésimo, no dice nada)
  - ❌ "se corrigió el error" (Voz pasiva, muy robótico)
  - ✅ "fix(cli): corrijo panic al no tener config" (Bien)

  # FORMATO DE SALIDA ESTRICTO
  ⚠️ CRÍTICO: DEBES devolver SOLO JSON válido. Sin bloques de markdown, sin explicaciones, sin texto antes/después.
  ⚠️ TODOS los tipos de campos están ESTRICTAMENTE definidos. NO cambies tipos ni agregues campos extra.
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.
  
  ## Schema JSON (OBLIGATORIO):
  {
    "type": "array",
    "items": {
      "type": "object",
      "required": ["title", "desc", "files", "analysis"],
      "properties": {
        "title": {
          "type": "string",
          "description": "Título del commit (tipo(scope): mensaje)"
        },
        "desc": {
          "type": "string",
          "description": "Explicación detallada en primera persona"
        },
        "files": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Array de rutas de archivos como strings"
        },
        "analysis": {
          "type": "object",
          "required": ["overview", "purpose", "impact"],
          "properties": {
            "overview": {"type": "string"},
            "purpose": {"type": "string"},
            "impact": {"type": "string"}
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    }
  }

  ## Reglas de Tipos (ESTRICTAS):
  - "title": DEBE ser string (nunca número, nunca null)
  - "desc": DEBE ser string (nunca número, nunca null, puede ser "" si está vacío)
  - "files": DEBE ser array de strings (nunca array de números, nunca null)
  - "analysis.overview": DEBE ser string
  - "analysis.purpose": DEBE ser string
  - "analysis.impact": DEBE ser string

  ## Acciones Prohibidas:
  ❌ NO agregues campos que no estén en el schema
  ❌ NO cambies tipos de campos (ej: desc a número)
  ❌ NO envuelvas el JSON en bloques de markdown
  ❌ NO agregues texto explicativo antes/después del JSON
  ❌ NO uses null para campos string requeridos (usa "" en su lugar)

  ## Ejemplo Válido:
  [
    {
      "title": "fix(cli): corrijo panic al no tener config",
      "desc": "Agregué validación para evitar panic cuando no hay archivo de configuración",
      "files": ["internal/config/config.go"],
      "analysis": {
        "overview": "Agregué validación de configuración",
        "purpose": "Prevenir panic cuando falta config",
        "impact": "Mejora la robustez del CLI"
      }
    }
  ]

  {{.TechnicalInfo}}

  Genera {{.Count}} sugerencias ahora. Devuelve SOLO el array JSON, nada más.`

	promptTemplateWithoutTicketEN = `# Task
  Act as a Git Specialist and generate {{.Count}} commit message suggestions based on code changes.

  # Inputs
  - Modified Files: {{.Files}}
  - Code Changes (Diff): {{.Diff}}
  - Issue Instructions: {{.Instructions}}
  - Recent History: {{.History}}

  # Generation Strategy
  1. **Analyze Diff:** Identify logic changes vs formatting.
  2. **Categorize:**
     - New feature? -> feat
     - Bug fix? -> fix
     - Code change without logic change? -> refactor
     - Docs only? -> docs
  3. **Drafting:**
     - Title: Imperative mood, max 50 chars if possible (e.g., "add validation", not "adding").
     - Description: First person, professional tone. "I added this validation to prevent X error".

  # Style Examples
  - ❌ "update main.go" (Terrible, says nothing)
  - ❌ "error was fixed" (Passive voice)
  - ✅ "fix(cli): handle panic when config is missing" (Perfect)

  # STRICT OUTPUT FORMAT
  ⚠️ CRITICAL: You MUST return ONLY valid JSON. No markdown blocks, no explanations, no text before/after.
  ⚠️ ALL field types are STRICTLY enforced. DO NOT change types or add extra fields.
  
  ## JSON Schema (MANDATORY):
  {
    "type": "array",
    "items": {
      "type": "object",
      "required": ["title", "desc", "files", "analysis"],
      "properties": {
        "title": {
          "type": "string",
          "description": "Commit title (type(scope): message)"
        },
        "desc": {
          "type": "string",
          "description": "Detailed explanation in first person"
        },
        "files": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Array of file paths as strings"
        },
        "analysis": {
          "type": "object",
          "required": ["overview", "purpose", "impact"],
          "properties": {
            "overview": {"type": "string"},
            "purpose": {"type": "string"},
            "impact": {"type": "string"}
          },
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    }
  }

  ## Type Rules (STRICT):
  - "title": MUST be string (never number, never null)
  - "desc": MUST be string (never number, never null, can be empty string "")
  - "files": MUST be array of strings (never array of numbers, never null)
  - "analysis.overview": MUST be string
  - "analysis.purpose": MUST be string
  - "analysis.impact": MUST be string

  ## Prohibited Actions:
  ❌ DO NOT add any fields not listed in the schema
  ❌ DO NOT change field types (e.g., desc to number)
  ❌ DO NOT wrap JSON in markdown code blocks
  ❌ DO NOT add explanatory text before/after JSON
  ❌ DO NOT use null values for required string fields (use "" instead)

  ## Valid Example:
  [
    {
      "title": "fix(cli): handle panic when config is missing",
      "desc": "I added validation to prevent panic when configuration file is missing",
      "files": ["internal/config/config.go"],
      "analysis": {
        "overview": "Added configuration validation",
        "purpose": "Prevent panic when config is missing",
        "impact": "Improves CLI robustness"
      }
    }
  ]

  {{.TechnicalInfo}}

  Generate {{.Count}} suggestions now. Return ONLY the JSON array, nothing else.`
)

const (
	releasePromptTemplateES = `# Tarea
Generá release notes profesionales para un CHANGELOG.md siguiendo el estándar "Keep a Changelog".

# Datos del Release
- Repo: {{.RepoOwner}}/{{.RepoName}}
- Versiones: {{.CurrentVersion}} -> {{.LatestVersion}} ({{.ReleaseDate}})

# Changelog (Diff)
{{.Changelog}}

# Instrucciones Críticas

## 1. FILTRADO DE RUIDO TÉCNICO
**IGNORAR completamente** estos tipos de commits (no incluirlos en ninguna sección):
- Cambios en mocks o tests internos (ej: "Implementa GetIssue en MockVCSClient")
- Refactors internos que no afectan funcionalidad (ej: "Refactor: extract helper function")
- Updates menores de dependencias (ej: "chore: update go.mod")
- Cambios de documentación interna o comentarios
- Fixes de typos en código o variables internas

**SÍ INCLUIR** solo cambios que impactan al usuario final:
- Nuevas features visibles
- Mejoras de performance o UX
- Correcciones de bugs que afectaban funcionalidad
- Breaking changes
- Updates importantes de dependencias (cambios de versión mayor)

## 2. AGRUPACIÓN INTELIGENTE
**AGRUPAR** commits relacionados bajo un concepto unificador:

❌ **MAL** (lista cruda de commits):
- "feat: agregar spinners"
- "feat: agregar colores"
- "feat: mejorar feedback visual"

✅ **BIEN** (agrupado con valor):
- "UX Renovada: Agregamos spinners, colores y feedback visual en todas las operaciones largas para que no sientas que la terminal se colgó"

**Reglas de agrupación:**
- Si 3+ commits tocan el mismo módulo/feature → agrupar en un solo highlight
- Priorizar el VALOR para el usuario, no los detalles técnicos
- Máximo 5-7 highlights por release (no listar 15+ ítems)

## 3. IDIOMA Y TONO
**ESPAÑOL ARGENTINO PROFESIONAL:**
- Tono: Conversacional pero técnico, como un email entre devs
- Primera persona plural: "Agregamos", "Mejoramos", "Implementamos"
- Evitar spanglish completamente (nada de "fixeamos" o "pusheamos")
- Evitar jerga forzada, mantener profesionalismo

**Ejemplos de tono correcto:**
- ✅ "Automatizamos la generación del CHANGELOG.md"
- ✅ "Mejoramos la detección automática de issues"
- ❌ "Se implementó la feature de changelog" (muy formal/pasivo)
- ❌ "Agregamos un fix re-copado" (muy informal)

## 4. ESTRUCTURA Y NARRATIVA
Cada release debe contar una historia:
- **Summary:** Explicar el foco principal del release (ej: "En esta versión nos enfocamos en mejorar la UX y automatizar el proceso de releases")
- **Highlights:** Agrupar por tema (UX, Automatización, Performance, etc.)
- Cada highlight debe responder: "¿Qué ganó el usuario con esto?"

## 5. FORMATO DE SALIDA ESTRICTO
⚠️ CRÍTICO: DEBES devolver SOLO JSON válido. Sin bloques de markdown, sin explicaciones, sin texto antes/después.
⚠️ TODOS los tipos de campos están ESTRICTAMENTE definidos. NO cambies tipos ni agregues campos extra.

## Schema JSON (OBLIGATORIO):
{
  "type": "object",
  "required": ["title", "summary", "highlights", "breaking_changes", "contributors"],
  "properties": {
    "title": {
      "type": "string",
      "description": "Título conciso y descriptivo"
    },
    "summary": {
      "type": "string",
      "description": "2-3 oraciones explicando el foco del release en primera persona plural"
    },
    "highlights": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Array de highlights como strings"
    },
    "breaking_changes": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Array de breaking changes como strings (o [] si no hay)"
    },
    "contributors": {
      "type": "string",
      "description": "Texto con contribuidores (ej: 'Gracias a @user1, @user2') o 'N/A'"
    }
  },
  "additionalProperties": false
}

## Reglas de Tipos (ESTRICTAS):
- "title": DEBE ser string (nunca número, nunca null)
- "summary": DEBE ser string (nunca número, nunca null)
- "highlights": DEBE ser array de strings (nunca array de números, nunca null, usar [] si está vacío)
- "breaking_changes": DEBE ser array de strings (nunca array de números, nunca null, usar [] si no hay)
- "contributors": DEBE ser string (nunca número, nunca null, usar "N/A" si no hay contribuidores)

## Acciones Prohibidas:
❌ NO agregues campos que no estén en el schema
❌ NO cambies tipos de campos (ej: highlights a objeto)
❌ NO envuelvas el JSON en bloques de markdown
❌ NO agregues texto explicativo antes/después del JSON
❌ NO uses null para campos requeridos (usa [] para arrays vacíos, "N/A" para contributors vacío)

## Ejemplo Válido:
{
  "title": "Mejoras de Experiencia de Usuario",
  "summary": "En esta versión nos enfocamos en mejorar la experiencia de usuario agregando feedback visual completo. Ya no vas a sentir que la terminal se colgó durante operaciones largas.",
  "highlights": [
    "UX Renovada: Agregamos spinners, colores y feedback visual en todas las operaciones largas (#45)",
    "Correcciones: Mejoramos el formato de los spinners para mejor legibilidad"
  ],
  "breaking_changes": [],
  "contributors": "N/A"
}

Generá las release notes ahora siguiendo estas instrucciones al pie de la letra.`

	releasePromptTemplateEN = `# Task
Generate professional release notes for a CHANGELOG.md following the "Keep a Changelog" standard.

# Release Information
- Repository: {{.RepoOwner}}/{{.RepoName}}
- Versions: {{.CurrentVersion}} -> {{.LatestVersion}} ({{.ReleaseDate}})

# Changelog (Diff)
{{.Changelog}}

# Critical Instructions

## 1. TECHNICAL NOISE FILTERING
**COMPLETELY IGNORE** these types of commits (do not include them in any section):
- Changes to mocks or internal tests (e.g., "Implement GetIssue in MockVCSClient")
- Internal refactors that don't affect functionality (e.g., "Refactor: extract helper function")
- Minor dependency updates (e.g., "chore: update go.mod")
- Internal documentation or comment changes
- Typo fixes in code or internal variables

**DO INCLUDE** only changes that impact the end user:
- New visible features
- Performance or UX improvements
- Bug fixes affecting functionality
- Breaking changes
- Important dependency updates (major version changes)

## 2. INTELLIGENT GROUPING
**GROUP** related commits under a unifying concept:

❌ **BAD** (raw commit list):
- "feat: add spinners"
- "feat: add colors"
- "feat: improve visual feedback"

✅ **GOOD** (grouped with value):
- "Revamped UX: Added spinners, colors, and visual feedback across all long-running operations so you never feel like the terminal froze"

**Grouping rules:**
- If 3+ commits touch the same module/feature → group into a single highlight
- Prioritize USER VALUE, not technical details
- Maximum 5-7 highlights per release (don't list 15+ items)

## 3. LANGUAGE AND TONE
**PROFESSIONAL ENGLISH:**
- Tone: Conversational yet technical, like an email between developers
- First person plural: "We added", "We improved", "We implemented"
- Maintain professionalism, avoid forced slang

**Examples of correct tone:**
- ✅ "We automated CHANGELOG.md generation"
- ✅ "We improved automatic issue detection"
- ❌ "The changelog feature was implemented" (too formal/passive)
- ❌ "We added a super cool fix" (too informal)

## 4. STRUCTURE AND NARRATIVE
Each release should tell a story:
- **Summary:** Explain the main focus of the release (e.g., "In this release, we focused on improving UX and automating the release process")
- **Highlights:** Group by theme (UX, Automation, Performance, etc.)
- Each highlight should answer: "What did the user gain from this?"

## 5. STRICT OUTPUT FORMAT
⚠️ CRITICAL: You MUST return ONLY valid JSON. No markdown blocks, no explanations, no text before/after.
⚠️ ALL field types are STRICTLY enforced. DO NOT change types or add extra fields.

## JSON Schema (MANDATORY):
{
  "type": "object",
  "required": ["title", "summary", "highlights", "breaking_changes", "contributors"],
  "properties": {
    "title": {
      "type": "string",
      "description": "Concise and descriptive title"
    },
    "summary": {
      "type": "string",
      "description": "2-3 sentences explaining the release focus in first person plural"
    },
    "highlights": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Array of highlight strings"
    },
    "breaking_changes": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "Array of breaking change strings (or [] if none)"
    },
    "contributors": {
      "type": "string",
      "description": "Contributors text (e.g., 'Thanks to @user1, @user2') or 'N/A'"
    }
  },
  "additionalProperties": false
}

## Type Rules (STRICT):
- "title": MUST be string (never number, never null)
- "summary": MUST be string (never number, never null)
- "highlights": MUST be array of strings (never array of numbers, never null, use [] if empty)
- "breaking_changes": MUST be array of strings (never array of numbers, never null, use [] if none)
- "contributors": MUST be string (never number, never null, use "N/A" if no contributors)

## Prohibited Actions:
❌ DO NOT add any fields not listed in the schema
❌ DO NOT change field types (e.g., highlights to object)
❌ DO NOT wrap JSON in markdown code blocks
❌ DO NOT add explanatory text before/after JSON
❌ DO NOT use null values for required fields (use [] for empty arrays, "N/A" for empty contributors)

## Valid Example:
{
  "title": "User Experience Improvements",
  "summary": "In this release, we focused on improving the user experience by adding complete visual feedback. You'll no longer feel like the terminal froze during long operations.",
  "highlights": [
    "Revamped UX: Added spinners, colors, and visual feedback across all long-running operations (#45)",
    "Fixes: Improved spinner formatting for better readability"
  ],
  "breaking_changes": [],
  "contributors": "N/A"
}

Generate the release notes now following these instructions to the letter.`
)

// GetPRPromptTemplate returns the appropriate template based on the language
func GetPRPromptTemplate(lang string) string {
	switch lang {
	case "es":
		return prPromptTemplateES
	default:
		return prPromptTemplateEN
	}
}

// GetCommitPromptTemplate returns the commit template based on language and whether there is a ticket
func GetCommitPromptTemplate(lang string, hasTicket bool) string {
	switch {
	case lang == "es" && hasTicket:
		return promptTemplateWithTicketES
	case lang == "es" && !hasTicket:
		return promptTemplateWithoutTicketES
	case hasTicket:
		return promptTemplateWithTicketEN
	default:
		return promptTemplateWithoutTicketEN
	}
}

// GetReleasePromptTemplate returns the release template based on the language
func GetReleasePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return releasePromptTemplateES
	default:
		return releasePromptTemplateEN
	}
}

// GetIssueReferenceInstructions returns issue reference instructions based on the language
func GetIssueReferenceInstructions(lang string) string {
	switch lang {
	case "es":
		return issueReferenceInstructionsES
	default:
		return issueReferenceInstructionsEN
	}
}

const (
	templateInstructionsES = `## Template del Proyecto

El proyecto tiene un template específico. DEBES seguir su estructura y formato al generar el contenido.

IMPORTANTE: Generá el contenido siguiendo la estructura y formato mostrado en el template arriba. Completá cada sección basándote en los cambios de código y el contexto proporcionado.

⚠️ CRÍTICO: A pesar del template arriba, tu respuesta DEBE SER JSON válido siguiendo el schema exacto definido en este prompt. El contenido del template debe incorporarse en el campo "description" como texto markdown, pero la respuesta general DEBE ser un objeto JSON con los campos "title", "description" y "labels". NO generes markdown o prosa - SOLO genera JSON válido.`

	templateInstructionsEN = `## Project Template

The project has a specific template. You MUST follow its structure and format when generating the content.

IMPORTANT: Generate the content following the structure and format shown in the template above. Fill in each section based on the code changes and context provided.

⚠️ CRITICAL: Despite the template above, your response MUST STILL be valid JSON following the exact schema defined in this prompt. The template content should be incorporated into the "description" field as markdown text, but the overall response MUST be a JSON object with "title", "description", and "labels" fields. Do NOT output markdown or prose - ONLY output valid JSON.`

	prTemplateInstructionsES = `## Template de PR del Proyecto

El proyecto tiene un template específico de PR. DEBES seguir su estructura y formato al generar la descripción del PR.

IMPORTANTE: Generá la descripción del PR siguiendo la estructura y formato mostrado en el template arriba. Completá cada sección basándote en los cambios de código y el contexto proporcionado.`

	prTemplateInstructionsEN = `## Project PR Template

The project has a specific PR template. You MUST follow its structure and format when generating the PR description.

IMPORTANT: Generate the PR description following the structure and format shown in the template above. Fill in each section based on the code changes and context provided.`
)

// GetTemplateInstructions returns template instructions based on the language
func GetTemplateInstructions(lang string) string {
	switch lang {
	case "es":
		return templateInstructionsES
	default:
		return templateInstructionsEN
	}
}

// GetPRTemplateInstructions returns PR template instructions based on the language
func GetPRTemplateInstructions(lang string) string {
	switch lang {
	case "es":
		return prTemplateInstructionsES
	default:
		return prTemplateInstructionsEN
	}
}

// FormatTemplateForPrompt formats a template for inclusion in an AI prompt.
// It handles both Issue and PR templates with proper language support.
func FormatTemplateForPrompt(template *models.IssueTemplate, lang string, templateType string) string {
	if template == nil {
		return ""
	}

	if lang == "" {
		lang = "en"
	}

	var sb strings.Builder
	isIssue := templateType == "issue"

	if lang == "es" {
		if isIssue {
			sb.WriteString("## Template de Issue del Proyecto\n\n")
			sb.WriteString("El proyecto tiene un template específico de issue. DEBES seguir su estructura y formato al generar el contenido del issue.\n\n")
		} else {
			sb.WriteString("## Template de PR del Proyecto\n\n")
			sb.WriteString("El proyecto tiene un template específico de PR. DEBES seguir su estructura y formato al generar la descripción del PR.\n\n")
		}
	} else {
		if isIssue {
			sb.WriteString("## Project Issue Template\n\n")
			sb.WriteString("The project has a specific issue template. You MUST follow its structure and format when generating the issue content.\n\n")
		} else {
			sb.WriteString("## Project PR Template\n\n")
			sb.WriteString("The project has a specific PR template. You MUST follow its structure and format when generating the PR description.\n\n")
		}
	}

	if template.Name != "" {
		if lang == "es" {
			sb.WriteString(fmt.Sprintf("Nombre del Template: %s\n", template.Name))
		} else {
			sb.WriteString(fmt.Sprintf("Template Name: %s\n", template.Name))
		}
	}

	if template.GetAbout() != "" {
		if lang == "es" {
			sb.WriteString(fmt.Sprintf("Descripción del Template: %s\n", template.GetAbout()))
		} else {
			sb.WriteString(fmt.Sprintf("Template Description: %s\n", template.GetAbout()))
		}
	}

	if template.BodyContent != "" {
		if lang == "es" {
			sb.WriteString("\nEstructura del Template:\n```markdown\n")
		} else {
			sb.WriteString("\nTemplate Structure:\n```markdown\n")
		}
		sb.WriteString(template.BodyContent)
		sb.WriteString("\n```\n\n")
		if isIssue {
			sb.WriteString(GetTemplateInstructions(lang))
		} else {
			sb.WriteString(GetPRTemplateInstructions(lang))
		}
		sb.WriteString("\n\n")
	} else if template.Body != nil {
		if lang == "es" {
			if isIssue {
				sb.WriteString("\nTipo de Template: GitHub Issue Form (YAML)\n")
			} else {
				sb.WriteString("\nTipo de Template: GitHub PR Template (YAML/Markdown)\n")
			}
			sb.WriteString("El template define campos específicos. Generá contenido que coincida con la estructura esperada.\n\n")
		} else {
			if isIssue {
				sb.WriteString("\nTemplate Type: GitHub Issue Form (YAML)\n")
			} else {
				sb.WriteString("\nTemplate Type: GitHub PR Template (YAML/Markdown)\n")
			}
			sb.WriteString("The template defines specific fields. Generate content that matches the expected structure.\n\n")
		}
	}

	return sb.String()
}

const (
	prIssueContextInstructionsES = `
  **IMPORTANTE - Contexto de Issues/Tickets:**
  Este PR está relacionado con los siguientes issues:
  {{.RelatedIssues}}

  **INSTRUCCIONES CLAVES:**
  1. DEBES incluir AL INICIO del resumen (primeras líneas) las referencias de cierre:
     - Si resuelve bugs: "Fixes #N"
     - Si implementa features: "Closes #N"
     - Si solo relaciona: "Relates to #N"
     - Formato: "Closes #39, Fixes #41" (separados por comas)

  2. En la sección de cambios clave, menciona explícitamente cómo cada cambio impacta en el issue.

  3. Usa el formato correcto para que GitHub enlace los issues automáticamente.

  **Ejemplo de formato correcto:**
  Closes #39

  - **Primer cambio clave:**
    - Propósito: Resolver el problema reportado en #39...
    - Impacto técnico: ...
  `

	prIssueContextInstructionsEN = `
  **IMPORTANT - Issue/Ticket Context:**
  This PR is related to the following issues:
  {{.RelatedIssues}}

  **MANDATORY INSTRUCTIONS:**
  1. You MUST include at the BEGINNING of the summary (first lines) the closing references:
     - If fixing bugs: "Fixes #N"
     - If implementing features: "Closes #N"
     - If just relating: "Relates to #N"
     - Format: "Closes #39, Fixes #41" (comma separated)

  2. In the key changes section, explicitly mention how each change addresses the issue

  3. Use the correct format so GitHub auto-links the issues in the "Development" section

  **Example of correct format:**
  Closes #39

  - **First key change:**
    - Purpose: Resolve the problem reported in #39...
    - Technical impact: ...
  `
)

// GetPRIssueContextInstructions returns issue context instructions for PRs
func GetPRIssueContextInstructions(locale string) string {
	if locale == "es" {
		return prIssueContextInstructionsES
	}
	return prIssueContextInstructionsEN
}

// FormatIssuesForPrompt formats the issue list to be included in the prompt
func FormatIssuesForPrompt(issues []models.Issue, locale string) string {
	if len(issues) == 0 {
		return ""
	}

	var result strings.Builder
	for _, issue := range issues {
		if locale == "es" {
			result.WriteString(fmt.Sprintf("- Issue #%d: %s\n", issue.Number, issue.Title))
			if issue.Description != "" {
				desc := issue.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				result.WriteString(fmt.Sprintf("  Descripción: %s\n", desc))
			}
		} else {
			result.WriteString(fmt.Sprintf("- Issue #%d: %s\n", issue.Number, issue.Title))
			if issue.Description != "" {
				desc := issue.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				result.WriteString(fmt.Sprintf("  Description: %s\n", desc))
			}
		}
	}

	return result.String()
}

const (
	technicalAnalysisES = `Proporciona un análisis técnico detallado incluyendo: buenas prácticas aplicadas, impacto en rendimiento/mantenibilidad, y consideraciones de seguridad si aplican.`
	technicalAnalysisEN = `Provide detailed technical analysis including: best practices applied, performance/maintainability impact, and security considerations if applicable.`
)

func GetTechnicalAnalysisInstruction(locale string) string {
	if locale == "es" {
		return technicalAnalysisES
	}
	return technicalAnalysisEN
}

const (
	noIssueReferenceES = `No incluyas referencias de issues en el título.`
	noIssueReferenceEN = `Do not include issue references in the title.`
)

func GetNoIssueReferenceInstruction(locale string) string {
	if locale == "es" {
		return noIssueReferenceES
	}
	return noIssueReferenceEN
}

// Release Note Headers
var (
	releaseHeadersES = map[string]string{
		"breaking":      "CAMBIOS QUE ROMPEN:",
		"features":      "NUEVAS CARACTERÍSTICAS:",
		"fixes":         "CORRECCIONES DE BUGS:",
		"improvements":  "MEJORAS:",
		"closed_issues": "ISSUES CERRADOS:",
		"merged_prs":    "PULL REQUESTS MERGEADOS:",
		"contributors":  "CONTRIBUIDORES",
		"file_stats":    "ESTADÍSTICAS DE ARCHIVOS:",
		"deps":          "ACTUALIZACIONES DE DEPENDENCIAS:",
	}

	releaseHeadersEN = map[string]string{
		"breaking":      "BREAKING CHANGES:",
		"features":      "NEW FEATURES:",
		"fixes":         "BUG FIXES:",
		"improvements":  "IMPROVEMENTS:",
		"closed_issues": "CLOSED ISSUES:",
		"merged_prs":    "MERGED PULL REQUESTS:",
		"contributors":  "CONTRIBUTORS",
		"file_stats":    "FILE STATISTICS:",
		"deps":          "DEPENDENCY UPDATES:",
	}
)

func GetReleaseNotesSectionHeaders(locale string) map[string]string {
	if locale == "es" {
		return releaseHeadersES
	}
	return releaseHeadersEN
}

const (
	issuePromptTemplateEN = `# STRICT OUTPUT FORMAT
  ⚠️ CRITICAL: You MUST return ONLY valid JSON. No markdown blocks, no explanations, no text before/after.
  ⚠️ ALL field types are STRICTLY enforced. DO NOT change types or add extra fields.
  ⚠️ DO NOT RETURN AN ARRAY. You MUST return a JSON OBJECT with exactly these fields: title, description, labels.

  ## JSON Schema (MANDATORY):
  {
    "type": "object",
    "required": ["title", "description", "labels"],
    "properties": {
      "title": {
        "type": "string",
        "description": "Concise and descriptive title"
      },
      "description": {
        "type": "string",
        "description": "Markdown body following the structure: Context, Technical Details, Impact"
      },
      "labels": {
        "type": "array",
        "items": {
          "type": "string"
        },
        "description": "Array of label strings (feature, fix, refactor, docs, test, infra)"
      }
    },
    "additionalProperties": false
  }

  ## Type Rules (STRICT):
  - "title": MUST be string (never number, never null, never empty)
  - "description": MUST be string (never number, never null, can contain markdown)
  - "labels": MUST be array of strings (never array of numbers, never null, use [] if empty)

  ## Prohibited Actions:
  ❌ DO NOT return an array like []
  ❌ DO NOT add any fields not listed in the schema
  ❌ DO NOT change field types (e.g., title to number)
  ❌ DO NOT wrap JSON in markdown code blocks
  ❌ DO NOT add explanatory text before/after JSON
  ❌ DO NOT use null values for required fields

  ## Valid Example:
  {
    "title": "feat: implement user authentication",
    "description": "### Context\nWe need user authentication to secure the application.\n\n### Technical Details\n- Added auth models\n- Implemented JWT tokens\n\n### Impact\nUsers can now securely access the application.",
    "labels": ["feature", "auth"]
  }

  # Task
  Act as a Senior Tech Lead and generate a high-quality GitHub issue based on the provided inputs.

  # Inputs
  {{.IssueInfo}}

  # Golden Rules (Constraints)
  1. **Active Voice:** Write in FIRST PERSON ("I implemented", "I added", "We refactored"). Avoid passive voice like "It was implemented".
  2. **Context First:** Explain the WHY before the WHAT.
  3. **Accurate Categorization:** Always choose at least one primary category: 'feature', 'fix', or 'refactor'. Use 'fix' ONLY for bug corrections. Use 'refactor' for code improvements without logic changes. Use 'feature' for new functionality.
  4. **No Emojis:** Do not use emojis in the title or description. Keep it purely textual and professional.
  5. **Balanced Labeling:** Aim for 2-4 relevant labels. Ensure you include the primary category plus any relevant file-based labels like 'test', 'docs', or 'infra' if applicable.
  6. **Format:** Raw JSON only. Do not wrap in markdown blocks.

  # Description Structure
  The 'description' field must follow this Markdown structure:
  - ### Context (Motivation)
  - ### Technical Details (Architectural changes, new models, etc.)
  - ### Impact (Benefits)

  Generate the issue now. Return ONLY the JSON object (NOT an array), nothing else.`

	issuePromptTemplateES = `# FORMATO DE SALIDA ESTRICTO
  ⚠️ CRÍTICO: DEBES devolver SOLO JSON válido. Sin bloques de markdown, sin explicaciones, sin texto antes/después.
  ⚠️ TODOS los tipos de campos están ESTRICTAMENTE definidos. NO cambies tipos ni agregues campos extra.
  ⚠️ NO DEVUELVAS UN ARRAY. DEBES devolver un OBJETO JSON con exactamente estos campos: title, description, labels.
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

  ## Schema JSON (OBLIGATORIO):
  {
    "type": "object",
    "required": ["title", "description", "labels"],
    "properties": {
      "title": {
        "type": "string",
        "description": "Título descriptivo y con gancho"
      },
      "description": {
        "type": "string",
        "description": "Cuerpo en markdown siguiendo la estructura: Contexto, Detalles Técnicos, Impacto"
      },
      "labels": {
        "type": "array",
        "items": {
          "type": "string"
        },
        "description": "Array de etiquetas como strings (feature, fix, refactor, docs, test, infra)"
      }
    },
    "additionalProperties": false
  }

  ## Reglas de Tipos (ESTRICTAS):
  - "title": DEBE ser string (nunca número, nunca null, nunca vacío)
  - "description": DEBE ser string (nunca número, nunca null, puede contener markdown)
  - "labels": DEBE ser array de strings (nunca array de números, nunca null, usar [] si está vacío)

  ## Acciones Prohibidas:
  ❌ NO devuelvas un array como []
  ❌ NO agregues campos que no estén en el schema
  ❌ NO cambies tipos de campos (ej: title a número)
  ❌ NO envuelvas el JSON en bloques de markdown
  ❌ NO agregues texto explicativo antes/después del JSON
  ❌ NO uses null para campos requeridos

  ## Ejemplo Válido:
  {
    "title": "feat: implementar autenticación de usuarios",
    "description": "### Contexto\nNecesitamos autenticación de usuarios para asegurar la aplicación.\n\n### Detalles Técnicos\n- Agregué modelos de auth\n- Implementé tokens JWT\n\n### Impacto\nLos usuarios ahora pueden acceder de forma segura a la aplicación.",
    "labels": ["feature", "auth"]
  }

  # Tarea
  Actuá como un Tech Lead y generá un issue de GitHub profesional basado en los inputs.

  # Entradas (Inputs)
  {{.IssueInfo}}

  # Reglas de Oro (Constraints)
  1. **Voz Activa:** Escribí en PRIMERA PERSONA ("Implementé", "Agregué", "Corregí"). Prohibido usar voz pasiva robótica.
  2. **Contexto Real:** Explicá el POR QUÉ del cambio, no solo qué líneas tocaste.
  3. **Categorización Precisa:** Elegí siempre al menos una categoría principal: 'feature', 'fix', o 'refactor'. Solo usá 'fix' si ves una corrección de un bug. Usá 'refactor' para mejoras de código sin cambios lógicos. Usá 'feature' para funcionalidades nuevas.
  4. **Cero Emojis:** No uses emojis ni en el título ni en el cuerpo del issue. Mantené un estilo sobrio y técnico.
  5. **Etiquetado Equilibrado:** Buscá entre 2 y 4 etiquetas relevantes. Asegurate de incluir la categoría principal más cualquier etiqueta de tipo de archivo como 'test', 'docs', o 'infra' si corresponde.
  6. **Formato:** JSON crudo. No incluyas bloques de markdown (como ` + "" + `).

  # Estructura de la Descripción
  El campo "description" tiene que ser Markdown y seguir esta estructura estricta:
  - ### Contexto (¿Cuál es la motivación o el dolor que resuelve esto?)
  - ### Detalles Técnicos (Lista de cambios importantes, modelos nuevos, refactors)
  - ### Impacto (¿Qué gana el usuario o el desarrollador con esto?)

  Generá el issue ahora. Devuelve SOLO el objeto JSON (NO un array), nada más.`
)

// GetIssuePromptTemplate returns the appropriate issue generation template based on language
func GetIssuePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return issuePromptTemplateES
	default:
		return issuePromptTemplateEN
	}
}
