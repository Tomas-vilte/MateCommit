package models

type (
	// PRData contiene la información extraída de una Pull Request.
	PRData struct {
		ID      int
		Creator string
		Commits []Commit
		Diff    string
	}

	// Commit representa un commit incluido en el PR.
	Commit struct {
		Message string
	}

	// PRSummary es el resumen generado para el PR, con título, cuerpo y etiquetas.
	PRSummary struct {
		Title  string
		Body   string
		Labels []string
	}
)
